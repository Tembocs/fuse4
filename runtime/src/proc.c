/*
 * proc.c — Process, environment, and time surface.
 */

#include "fuse_rt.h"
#include <stdlib.h>

/* Stored by fuse_rt_init(). */
static int    g_argc = 0;
static char **g_argv = NULL;

void fuse_rt_init(int argc, char **argv) {
    g_argc = argc;
    g_argv = argv;
}

int fuse_rt_proc_argc(void) {
    return g_argc;
}

const char *fuse_rt_proc_argv(int index) {
    if (index < 0 || index >= g_argc) return NULL;
    return g_argv[index];
}

const char *fuse_rt_proc_env_get(const char *name) {
    return getenv(name);
}

_Noreturn void fuse_rt_proc_exit(int code) {
    exit(code);
}
