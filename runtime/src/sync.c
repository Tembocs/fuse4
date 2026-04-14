/*
 * sync.c — Mutex and condition variable surface.
 */

#include "fuse_rt.h"

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>

struct fuse_rt_mutex {
    CRITICAL_SECTION cs;
};

fuse_rt_mutex *fuse_rt_mutex_create(void) {
    fuse_rt_mutex *m = (fuse_rt_mutex *)fuse_rt_mem_alloc(sizeof(fuse_rt_mutex));
    InitializeCriticalSection(&m->cs);
    return m;
}

void fuse_rt_mutex_lock(fuse_rt_mutex *m) {
    EnterCriticalSection(&m->cs);
}

void fuse_rt_mutex_unlock(fuse_rt_mutex *m) {
    LeaveCriticalSection(&m->cs);
}

void fuse_rt_mutex_destroy(fuse_rt_mutex *m) {
    DeleteCriticalSection(&m->cs);
    fuse_rt_mem_free(m);
}

struct fuse_rt_cond {
    CONDITION_VARIABLE cv;
};

fuse_rt_cond *fuse_rt_cond_create(void) {
    fuse_rt_cond *c = (fuse_rt_cond *)fuse_rt_mem_alloc(sizeof(fuse_rt_cond));
    InitializeConditionVariable(&c->cv);
    return c;
}

void fuse_rt_cond_wait(fuse_rt_cond *c, fuse_rt_mutex *m) {
    SleepConditionVariableCS(&c->cv, &m->cs, INFINITE);
}

void fuse_rt_cond_signal(fuse_rt_cond *c) {
    WakeConditionVariable(&c->cv);
}

void fuse_rt_cond_broadcast(fuse_rt_cond *c) {
    WakeAllConditionVariable(&c->cv);
}

void fuse_rt_cond_destroy(fuse_rt_cond *c) {
    fuse_rt_mem_free(c);
}

#else
#include <pthread.h>

struct fuse_rt_mutex {
    pthread_mutex_t mtx;
};

fuse_rt_mutex *fuse_rt_mutex_create(void) {
    fuse_rt_mutex *m = (fuse_rt_mutex *)fuse_rt_mem_alloc(sizeof(fuse_rt_mutex));
    pthread_mutex_init(&m->mtx, NULL);
    return m;
}

void fuse_rt_mutex_lock(fuse_rt_mutex *m) {
    pthread_mutex_lock(&m->mtx);
}

void fuse_rt_mutex_unlock(fuse_rt_mutex *m) {
    pthread_mutex_unlock(&m->mtx);
}

void fuse_rt_mutex_destroy(fuse_rt_mutex *m) {
    pthread_mutex_destroy(&m->mtx);
    fuse_rt_mem_free(m);
}

struct fuse_rt_cond {
    pthread_cond_t cv;
};

fuse_rt_cond *fuse_rt_cond_create(void) {
    fuse_rt_cond *c = (fuse_rt_cond *)fuse_rt_mem_alloc(sizeof(fuse_rt_cond));
    pthread_cond_init(&c->cv, NULL);
    return c;
}

void fuse_rt_cond_wait(fuse_rt_cond *c, fuse_rt_mutex *m) {
    pthread_cond_wait(&c->cv, &m->mtx);
}

void fuse_rt_cond_signal(fuse_rt_cond *c) {
    pthread_cond_signal(&c->cv);
}

void fuse_rt_cond_broadcast(fuse_rt_cond *c) {
    pthread_cond_broadcast(&c->cv);
}

void fuse_rt_cond_destroy(fuse_rt_cond *c) {
    pthread_cond_destroy(&c->cv);
    fuse_rt_mem_free(c);
}

#endif
