/* SPDX-License-Identifier: BSD-3-Clause */
/* Disposable experiment: kernel identity beats a forged payload UID. */
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/ucred.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <err.h>
#include <pwd.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

static void
probe(const char *path, uid_t uid, gid_t gid)
{
    struct sockaddr_un address = { .sun_family = AF_UNIX };
    struct xucred credential;
    socklen_t length = sizeof(credential);
    int listener, connection, status;
    pid_t child;
    uid_t claimed = 0;

    if (strlcpy(address.sun_path, path, sizeof(address.sun_path)) >= sizeof(address.sun_path))
        errx(1, "fixture socket path too long");
    address.sun_len = sizeof(address);
    if ((listener = socket(AF_UNIX, SOCK_STREAM, 0)) == -1 ||
        bind(listener, (struct sockaddr *)&address, sizeof(address)) == -1 ||
        chmod(path, 0666) == -1 || listen(listener, 1) == -1)
        err(1, "listen");
    if ((child = fork()) == -1)
        err(1, "fork");
    if (child == 0) {
        close(listener);
        if (setgroups(0, NULL) == -1 || setgid(gid) == -1 || setuid(uid) == -1)
            err(1, "drop identity before connect");
        if ((connection = socket(AF_UNIX, SOCK_STREAM, 0)) == -1 ||
            connect(connection, (struct sockaddr *)&address, sizeof(address)) == -1 ||
            write(connection, &claimed, sizeof(claimed)) != sizeof(claimed))
            err(1, "connect/write");
        close(connection);
        _exit(0);
    }
    if ((connection = accept(listener, NULL, NULL)) == -1 ||
        getsockopt(connection, SOL_LOCAL, LOCAL_PEERCRED, &credential, &length) == -1)
        err(1, "accept/LOCAL_PEERCRED");
    if (length != sizeof(credential) || credential.cr_version != XUCRED_VERSION ||
        credential.cr_uid != uid)
        errx(1, "kernel peer credential mismatch");
    if (read(connection, &claimed, sizeof(claimed)) != sizeof(claimed) || claimed != 0)
        errx(1, "expected forged root claim");
    if (waitpid(child, &status, 0) == -1 || !WIFEXITED(status) || WEXITSTATUS(status) != 0)
        errx(1, "client failed");
    printf("PASS: kernel UID %u; root-only policy %s despite payload UID 0\n",
        (unsigned)uid, credential.cr_uid == 0 ? "allow" : "deny");
    close(connection);
    close(listener);
    if (unlink(path) == -1)
        err(1, "unlink fixture socket");
}

int
main(int argc, char **argv)
{
    struct passwd *nobody;
    if (argc != 2 || geteuid() != 0)
        errx(1, "requires root and generated fixture socket path");
    if ((nobody = getpwnam("nobody")) == NULL || nobody->pw_uid == 0)
        errx(1, "missing unprivileged fixture identity");
    probe(argv[1], 0, 0);
    probe(argv[1], nobody->pw_uid, nobody->pw_gid);
    return 0;
}
