/*
 * nsenter(1) - command-line interface for setns(2)
 *
 * Copyright (C) 2012-2013 Eric Biederman <ebiederm@xmission.com>
 *
 * This program is free software; you can redistribute it and/or modify it
 * under the terms of the GNU General Public License as published by the
 * Free Software Foundation; version 2.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

#include <dirent.h>
#include <errno.h>
#include <getopt.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <unistd.h>
#include <assert.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <grp.h>

#include "strutils.h"
#include "nls.h"
#include "c.h"
#include "closestream.h"
#include "namespace.h"
#include "exec_shell.h"

static struct namespace_file {
	int nstype;
	const char *name;
	int fd;
} namespace_files[] = {
	/* Careful the order is significant in this array.
	 *
	 * The user namespace comes first, so that it is entered
	 * first.  This gives an unprivileged user the potential to
	 * enter the other namespaces.
	 */
	{ .nstype = CLONE_NEWNS,    .name = "ns/mnt",  .fd = -1 },
	{ .nstype = 0, .name = NULL, .fd = -1 }
};

static pid_t namespace_target_pid = 0;

static void open_target_fd(int *fd, const char *type, const char *path)
{
	char pathbuf[PATH_MAX];

	if (!path && namespace_target_pid) {
		snprintf(pathbuf, sizeof(pathbuf), "/proc/%u/%s",
			 namespace_target_pid, type);
		path = pathbuf;
	}
	if (!path)
		errx(EXIT_FAILURE,
		     _("neither filename nor target pid supplied for %s"),
		     type);

	if (*fd >= 0)
		close(*fd);

	*fd = open(path, O_RDONLY);
	if (*fd < 0)
		err(EXIT_FAILURE, _("cannot open %s"), path);
}

static void open_namespace_fd(int nstype, const char *path)
{
	struct namespace_file *nsfile;

	for (nsfile = namespace_files; nsfile->nstype; nsfile++) {
		if (nstype != nsfile->nstype)
			continue;

		open_target_fd(&nsfile->fd, nsfile->name, path);
		return;
	}
	/* This should never happen */
	assert(nsfile->nstype);
}

static void continue_as_child(void)
{
	pid_t child = fork();
	int status;
	pid_t ret;

	if (child < 0)
		err(EXIT_FAILURE, _("fork failed"));

	/* Only the child returns */
	if (child == 0)
		return;

	for (;;) {
		ret = waitpid(child, &status, WUNTRACED);
		if ((ret == child) && (WIFSTOPPED(status))) {
			/* The child suspended so suspend us as well */
			kill(getpid(), SIGSTOP);
			kill(child, SIGCONT);
		} else {
			break;
		}
	}
	/* Return the child's exit code if possible */
	if (WIFEXITED(status)) {
		exit(WEXITSTATUS(status));
	} else if (WIFSIGNALED(status)) {
		kill(getpid(), WTERMSIG(status));
	}
	exit(EXIT_FAILURE);
}

int main(int argc, char *argv[])
{
	static const struct option longopts[] = {
		{ "target", required_argument, NULL, 't' },
		{ NULL, 0, NULL, 0 }
	};

	struct namespace_file *nsfile;
	int c;

	while ((c =
		getopt_long(argc, argv, "+hVt:m::u::i::n::p::C::U::S:G:r::w::FZ",
			    longopts, NULL)) != -1) {
		switch (c) {
		case 't':
			namespace_target_pid =
			    strtoul_or_err(optarg, _("failed to parse pid"));
			break;
		}
	}

	open_namespace_fd(CLONE_NEWNS, NULL);

	/*
	 * Now that we know which namespaces we want to enter, enter them.
	 */
	for (nsfile = namespace_files; nsfile->nstype; nsfile++) {
		if (nsfile->fd < 0)
			continue;
		if (setns(nsfile->fd, nsfile->nstype))
			err(EXIT_FAILURE,
			    _("reassociate to namespace '%s' failed"),
			    nsfile->name);
		close(nsfile->fd);
		nsfile->fd = -1;
	}

	if (optind < argc) {
		execvp(argv[optind], argv + optind);
		err(EXIT_FAILURE, _("failed to execute %s"), argv[optind]);
	}
	exec_shell();
}
