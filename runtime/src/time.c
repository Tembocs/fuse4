/*
 * time.c — Monotonic clock.
 */

#include "fuse_rt.h"

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>

int64_t fuse_rt_time_now_ns(void) {
    static LARGE_INTEGER freq = {0};
    LARGE_INTEGER counter;
    if (freq.QuadPart == 0) {
        QueryPerformanceFrequency(&freq);
    }
    QueryPerformanceCounter(&counter);
    /* Convert to nanoseconds: counter * 1e9 / freq */
    return (int64_t)((double)counter.QuadPart * 1e9 / (double)freq.QuadPart);
}

#else
#include <time.h>

int64_t fuse_rt_time_now_ns(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000000000LL + (int64_t)ts.tv_nsec;
}

#endif
