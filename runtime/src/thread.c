/*
 * thread.c — Thread spawn surface.
 */

#include "fuse_rt.h"

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>

typedef struct {
    fuse_rt_thread_fn fn;
    void *arg;
} thread_trampoline;

static DWORD WINAPI thread_entry(LPVOID param) {
    thread_trampoline *t = (thread_trampoline *)param;
    fuse_rt_thread_fn fn = t->fn;
    void *arg = t->arg;
    fuse_rt_mem_free(t);
    fn(arg);
    return 0;
}

int fuse_rt_thread_spawn(fuse_rt_thread_fn fn, void *arg) {
    thread_trampoline *t = (thread_trampoline *)fuse_rt_mem_alloc(sizeof(thread_trampoline));
    t->fn = fn;
    t->arg = arg;
    HANDLE h = CreateThread(NULL, 0, thread_entry, t, 0, NULL);
    if (!h) {
        fuse_rt_mem_free(t);
        return -1;
    }
    CloseHandle(h);
    return 0;
}

#else
#include <pthread.h>

typedef struct {
    fuse_rt_thread_fn fn;
    void *arg;
} thread_trampoline;

static void *thread_entry(void *param) {
    thread_trampoline *t = (thread_trampoline *)param;
    fuse_rt_thread_fn fn = t->fn;
    void *arg = t->arg;
    fuse_rt_mem_free(t);
    fn(arg);
    return NULL;
}

int fuse_rt_thread_spawn(fuse_rt_thread_fn fn, void *arg) {
    thread_trampoline *t = (thread_trampoline *)fuse_rt_mem_alloc(sizeof(thread_trampoline));
    t->fn = fn;
    t->arg = arg;
    pthread_t tid;
    int err = pthread_create(&tid, NULL, thread_entry, t);
    if (err != 0) {
        fuse_rt_mem_free(t);
        return -1;
    }
    pthread_detach(tid);
    return 0;
}

#endif
