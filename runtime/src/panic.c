/*
 * panic.c — Panic and abort primitives.
 */

#include "fuse_rt.h"
#include <stdio.h>
#include <stdlib.h>

_Noreturn void fuse_rt_panic(const char *message) {
    fprintf(stderr, "fuse panic: %s\n", message);
    abort();
}

_Noreturn void fuse_rt_abort(void) {
    abort();
}
