#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/time.h>

#define STRINGIFY(x) #x
#define VERSION_STRING(x) STRINGIFY(x)

#ifndef VERSION
#define VERSION HEAD
#endif

#define LOGFILE ("/proc/1/root/profile.log")
#define LOGPATTERN ("%s,%s,%d.%d\n")

#define PROPERTY_GET_PROFILE (-1)
#define PROPERTY_SET_PROFILE (1)
#define PROPERTY_SET_NO_PROFILE (0)

static int _profile(int profile) {
    static int optProfile = PROPERTY_SET_NO_PROFILE;
    if (profile != PROPERTY_GET_PROFILE) {
        optProfile = profile;
    }
    return optProfile;
}

static void sigdown(int signo) {
    psignal(signo, "Shutting down, got signal");
    exit(0);
}

int main(int argc, char **argv) {
    int i;
    for (i = 1; i < argc; ++i) {
        if (!strcasecmp(argv[i], "-p")) {
            _profile(PROPERTY_SET_PROFILE);
        }
    }

    if (_profile(PROPERTY_GET_PROFILE)) {
        FILE *logFile = fopen(LOGFILE, "a");
        struct timeval currentTime;

        gettimeofday(&currentTime, NULL);
        fprintf(logFile, LOGPATTERN, "sleep", "startup", (int)currentTime.tv_sec, (int)currentTime.tv_usec);

        fclose(logFile);
    }

    if (sigaction(SIGINT, &(struct sigaction){.sa_handler = sigdown}, NULL) < 0)
    return 1;
    if (sigaction(SIGTERM, &(struct sigaction){.sa_handler = sigdown}, NULL) < 0)
    return 2;

    for (;;)
        pause();
    fprintf(stderr, "Error: infinite loop terminated\n");
    return 42;
}
