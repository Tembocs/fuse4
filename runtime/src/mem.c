/*
 * mem.c — Memory allocation primitives.
 */

#include "fuse_rt.h"
#include <stdlib.h>
#include <stdio.h>

void *fuse_rt_mem_alloc(size_t size) {
    void *ptr = malloc(size);
    if (!ptr && size > 0) {
        fuse_rt_panic("fuse_rt_mem_alloc: out of memory");
    }
    return ptr;
}

void *fuse_rt_mem_alloc_zeroed(size_t count, size_t size) {
    void *ptr = calloc(count, size);
    if (!ptr && count > 0 && size > 0) {
        fuse_rt_panic("fuse_rt_mem_alloc_zeroed: out of memory");
    }
    return ptr;
}

void *fuse_rt_mem_realloc(void *ptr, size_t new_size) {
    void *new_ptr = realloc(ptr, new_size);
    if (!new_ptr && new_size > 0) {
        fuse_rt_panic("fuse_rt_mem_realloc: out of memory");
    }
    return new_ptr;
}

void fuse_rt_mem_free(void *ptr) {
    free(ptr);
}
