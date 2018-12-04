#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/time.h>
#include <arpa/inet.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <sys/socket.h>

#define STRINGIFY(x) #x
#define VERSION_STRING(x) STRINGIFY(x)

#ifndef VERSION
#define VERSION HEAD
#endif

#define PIDFILE ("/proc/1/root/%s.pid")
#define LOGFILE ("/proc/1/root/profile.log")
#define LOGPATTERN ("%s,%s,%d.%d\n")

#define PROPERTY_GET_PROFILE (-1)
#define PROPERTY_SET_PROFILE (1)
#define PROPERTY_SET_NO_PROFILE (0)

#define PORT 8080

#define ERROR_PID 3;
#define ERROR_MEMORY 4;
#define ERROR_SOCKET 11;
#define REQUEST_MESSAGE ("GET /%s HTTP/1.0\r\n\r\n")

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

int request(char *message);

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

    const char* fname = getenv("fname");
    int errcode = 0;
    if (fname != NULL) {
        char *faas = (char*)malloc(strlen(fname) + 1); // Add \0
        char *pid = (char*)malloc(strlen(PIDFILE) + strlen(fname) - 2 + 1); // Remove %s, Add \0
        char *message = (char*)malloc(strlen(REQUEST_MESSAGE) + strlen(fname) - 2 + 1); // Remove %s, Add \0
        if (faas != NULL && pid != NULL && message != NULL) {
            strcpy(faas, fname);
            sprintf(pid, PIDFILE, faas);
            sprintf(message, REQUEST_MESSAGE, faas);

            FILE *pidFile = fopen(pid, "w");
            printf("%s", pid);
            if (pidFile == NULL) {
                errcode = ERROR_PID;
            }
            else {
                const char* fprocess = getenv("fprocess");
                fprintf(pidFile, "%d\n%s", getpid(), fprocess != NULL ? fprocess : "");
                fclose(pidFile);

                errcode = request(message);
            }
        }
        if (faas != NULL) {
            free(faas);
        }
        if (pid != NULL) {
            free(pid);
        }
        if (message != NULL) {
            free(message);
        }
        if (errcode > 0) {
            return errcode;
        }
    }

    for (;;)
        pause();
    fprintf(stderr, "Error: infinite loop terminated\n");
    return 42;
}

int request(char *message)
{
    struct sockaddr_in dest;
    int sockfd, bytes, total;

    /* create the socket */
    sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) {
        return ERROR_SOCKET;
    }

    /* fill in the structure */
    memset(&dest, 0, sizeof(dest));
    dest.sin_family = AF_INET;
    dest.sin_addr.s_addr = htonl(INADDR_LOOPBACK); /* set destination IP number - localhost, 127.0.0.1*/
    dest.sin_port = htons(PORT);                   /* set destination port number */

    /* connect the socket, ignore if unable to connect */
    if (connect(sockfd, (struct sockaddr *)&dest, sizeof(dest)) < 0) {
        close(sockfd);
        return 0;
    }

    /* send the request */
    total = strlen(message);
    bytes = write(sockfd, message, total);
    if (bytes < 0) {
        close(sockfd);
        return ERROR_SOCKET;
    }

    /* close the socket with wait for response */
    shutdown (sockfd, 0); /* Stop receiving data for this socket. If further data arrives, reject it. */
    close(sockfd);

    return 0;
}
