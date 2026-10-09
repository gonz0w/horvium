/* SPDX-License-Identifier: BSD-3-Clause */
/* Disposable research observer; not a runtime adapter or clearance grant. */
#include <sys/types.h>
#include <sys/param.h>
#include <sys/proc.h>
#include <sys/sysctl.h>
#include <sys/user.h>
#include <sys/procctl.h>
#include <errno.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

int main(int argc, char **argv)
{
    struct timeval boot;
    struct kinfo_proc info;
    struct procctl_reaper_status reap;
    int mib[4], reaper_error = 0;
    size_t size;
    char *end;
    long pid;
    if (argc != 2 || getuid() != 0 || geteuid() != 0)
        return 2;
    errno = 0;
    pid = strtol(argv[1], &end, 10);
    if (errno || *argv[1] == '\0' || *end != '\0' || pid < 1 || pid > INT_MAX)
        return 2;
    size = sizeof(boot);
    if (sysctlbyname("kern.boottime", &boot, &size, NULL, 0) == -1 ||
        size != sizeof(boot) || boot.tv_sec < 1 || boot.tv_usec < 0 || boot.tv_usec >= 1000000)
        return 3;
    memset(&info, 0, sizeof(info));
    mib[0] = CTL_KERN; mib[1] = KERN_PROC; mib[2] = KERN_PROC_PID; mib[3] = (int)pid;
    size = sizeof(info);
    if (sysctl(mib, 4, &info, &size, NULL, 0) == -1) {
        if (errno != ESRCH)
            return 3;
        size = 0;
    }
    if (size != 0 && (size != sizeof(info) || info.ki_structsize != (int)sizeof(info) || info.ki_pid != pid))
        return 3;
    printf("{\"boot_sec\":%lld,\"boot_usec\":%ld,\"found\":%s,\"pid\":%ld",
        (long long)boot.tv_sec, (long)boot.tv_usec, size ? "true" : "false", pid);
    if (size) {
        memset(&reap, 0, sizeof(reap));
        if (procctl(P_PID, (id_t)pid, PROC_REAP_STATUS, &reap) == -1)
            reaper_error = errno;
        printf(",\"struct_bytes\":%zu,\"ppid\":%d,\"pgid\":%d,\"sid\":%d,"
            "\"start_sec\":%lld,\"start_usec\":%ld,\"uid\":%u,\"ruid\":%u,"
            "\"state\":%d,\"zombie\":%s,\"stopped\":%s,\"reaper_error\":%d,\"reaper\":%d,\"children\":%u,\"descendants\":%u,\"reaper_flags\":%u",
            sizeof(info), info.ki_ppid, info.ki_pgid, info.ki_sid,
            (long long)info.ki_start.tv_sec, (long)info.ki_start.tv_usec,
            info.ki_uid, info.ki_ruid, info.ki_stat, info.ki_stat == SZOMB ? "true" : "false", info.ki_stat == SSTOP ? "true" : "false", reaper_error,
            reap.rs_reaper, reap.rs_children, reap.rs_descendants, reap.rs_flags);
    }
    puts("}");
    return 0;
}
