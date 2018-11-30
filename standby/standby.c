#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/time.h>
#include <sys/wait.h>

#define STRINGIFY(x) #x
#define VERSION_STRING(x) STRINGIFY(x)

#ifndef VERSION
#define VERSION HEAD
#endif

#define PIDFILE ("/var/run/standby.pid")
#define LOGFILE ("/proc/1/root/profile.log")
#define LOGPATTERN ("%s,%s,%d.%d\n")

#define PROPERTY_GET_PROFILE (-1)
#define PROPERTY_SET_PROFILE (1)
#define PROPERTY_SET_NO_PROFILE (0)

#define TIMEOUT_ON_PROFILE 3

static int _profile(int profile) {
    static int optProfile = PROPERTY_SET_NO_PROFILE;
    if (profile != PROPERTY_GET_PROFILE) {
        optProfile = profile;
    }
    return optProfile;
}

static void _profiler(char *msg) {
    if (_profile(PROPERTY_GET_PROFILE)) {
        FILE *logFile = fopen(LOGFILE, "a");
        if (logFile == NULL) {
            fprintf(stderr, "Warn: failed to open %s\n", LOGFILE);
        }

        struct timeval currentTime;
        gettimeofday(&currentTime, NULL);
        fprintf(logFile, LOGPATTERN, "standby", msg, (int)currentTime.tv_sec, (int)currentTime.tv_usec);

        fclose(logFile);
    }
}

static void sigdown(int signo) {
    // Wait for possible child (eg:suicide) to exit;
    while (waitpid(-1, NULL, WNOHANG) > 0)
      ;

    _profiler("shutdown");
    psignal(signo, "Shutting down, got signal");
    // No need to clear pid, one time function
    exit(0);
}

static void sigalrm(int signo) {
    // Terminate on timeout
    sigdown(SIGINT);
}

int suicide(int sig) {
    int pid = 0;
    FILE *pidFile = fopen(PIDFILE, "r");
    if (pidFile == NULL) {
        return ESRCH;
    }

    fscanf(pidFile, "%d", &pid);
    fclose(pidFile);
    if (pid == 0) {
        return ESRCH;
    }

    return kill((pid_t)pid, sig);
}

int main(int argc, char **argv) {
    int i;
    int sig = 0;
    for (i = 1; i < argc; ++i) {
        if (!strcasecmp(argv[i], "-p")) {
            _profile(PROPERTY_SET_PROFILE);
            _profiler("startup");
            // Exit on timeout.
            if (TIMEOUT_ON_PROFILE > 0) {
                alarm(TIMEOUT_ON_PROFILE);
            }
        }
        else if (!strcasecmp(argv[i], "-s") && i < argc - 1) {
            sig = atoi(argv[i+1]);
            if (sig == 0) {
                sig = SIGINT;
            }
            return suicide(sig);
        }
    }

    FILE *pidFile = fopen(PIDFILE, "w");
    if (pidFile == NULL) {
        return 3;
    }
    fprintf(pidFile, "%d", getpid());
    fclose(pidFile);

    if (sigaction(SIGINT, &(struct sigaction){.sa_handler = sigdown}, NULL) < 0) {
        return 1;
    }
    if (sigaction(SIGTERM, &(struct sigaction){.sa_handler = sigdown}, NULL) < 0) {
        return 2;
    }
    if (sigaction(SIGALRM, &(struct sigaction){.sa_handler = sigalrm}, NULL) < 0) {
        return 3;
    }

    for (;;) {
        pause();
    }
    fprintf(stderr, "Error: infinite loop terminated\n");
    return 42;
}
