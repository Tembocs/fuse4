/*
 * io.c — Basic IO surface: stdout, stderr, files.
 */

#include "fuse_rt.h"
#include <stdio.h>

int64_t fuse_rt_io_write_stdout(const void *buf, size_t len) {
    size_t written = fwrite(buf, 1, len, stdout);
    return (int64_t)written;
}

int64_t fuse_rt_io_write_stderr(const void *buf, size_t len) {
    size_t written = fwrite(buf, 1, len, stderr);
    return (int64_t)written;
}

int64_t fuse_rt_io_file_open(const char *path, int mode) {
    const char *fmode;
    switch (mode) {
        case 0: fmode = "rb"; break;
        case 1: fmode = "wb"; break;
        case 2: fmode = "ab"; break;
        default: return -1;
    }
    FILE *f = fopen(path, fmode);
    if (!f) return -1;
    /* Store FILE* as int64_t handle. This is a bootstrap simplification. */
    return (int64_t)(uintptr_t)f;
}

int64_t fuse_rt_io_file_read(int64_t handle, void *buf, size_t len) {
    FILE *f = (FILE *)(uintptr_t)handle;
    if (!f) return -1;
    size_t n = fread(buf, 1, len, f);
    if (n == 0 && ferror(f)) return -1;
    return (int64_t)n;
}

int64_t fuse_rt_io_file_write(int64_t handle, const void *buf, size_t len) {
    FILE *f = (FILE *)(uintptr_t)handle;
    if (!f) return -1;
    size_t n = fwrite(buf, 1, len, f);
    return (int64_t)n;
}

void fuse_rt_io_file_close(int64_t handle) {
    FILE *f = (FILE *)(uintptr_t)handle;
    if (f) fclose(f);
}
