/*
 * fuse_rt.h — Fuse bootstrap runtime ABI.
 *
 * This header declares every entry point that compiled Fuse programs
 * may call at runtime. The naming convention is:
 *
 *   fuse_rt_{module}_{operation}
 *
 * The runtime is bootstrap infrastructure: small, explicit, and
 * reviewable. High-level behavior belongs in the Fuse stdlib.
 */

#ifndef FUSE_RT_H
#define FUSE_RT_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ===== Memory (Phase 01) ===== */

/* Allocate `size` bytes, abort on failure. */
void *fuse_rt_mem_alloc(size_t size);

/* Allocate `count * size` bytes zero-initialized, abort on failure. */
void *fuse_rt_mem_alloc_zeroed(size_t count, size_t size);

/* Reallocate `ptr` to `new_size` bytes, abort on failure. */
void *fuse_rt_mem_realloc(void *ptr, size_t new_size);

/* Free memory previously allocated by fuse_rt_mem_alloc*. */
void fuse_rt_mem_free(void *ptr);

/* ===== Panic and abort (Phase 01) ===== */

/* Print a panic message to stderr and abort. Does not return. */
_Noreturn void fuse_rt_panic(const char *message);

/* Abort without a message. Does not return. */
_Noreturn void fuse_rt_abort(void);

/* ===== IO (Phase 02) ===== */

/* Write `len` bytes from `buf` to stdout. Returns bytes written or -1. */
int64_t fuse_rt_io_write_stdout(const void *buf, size_t len);

/* Write `len` bytes from `buf` to stderr. Returns bytes written or -1. */
int64_t fuse_rt_io_write_stderr(const void *buf, size_t len);

/* Open a file. Returns a handle (>=0) or -1 on error.
 * mode: 0=read, 1=write, 2=append */
int64_t fuse_rt_io_file_open(const char *path, int mode);

/* Read up to `len` bytes into `buf`. Returns bytes read or -1. */
int64_t fuse_rt_io_file_read(int64_t handle, void *buf, size_t len);

/* Write `len` bytes from `buf`. Returns bytes written or -1. */
int64_t fuse_rt_io_file_write(int64_t handle, const void *buf, size_t len);

/* Close a file handle. */
void fuse_rt_io_file_close(int64_t handle);

/* ===== Process and environment (Phase 02) ===== */

/* Return the number of command-line arguments. */
int fuse_rt_proc_argc(void);

/* Return command-line argument at `index` (0-based), or NULL. */
const char *fuse_rt_proc_argv(int index);

/* Get environment variable value, or NULL if not set. */
const char *fuse_rt_proc_env_get(const char *name);

/* Exit the process with the given code. Does not return. */
_Noreturn void fuse_rt_proc_exit(int code);

/* ===== Time (Phase 02) ===== */

/* Return monotonic nanoseconds since an arbitrary epoch. */
int64_t fuse_rt_time_now_ns(void);

/* ===== Threads (Phase 03) ===== */

/* Thread function signature. */
typedef void (*fuse_rt_thread_fn)(void *arg);

/* Spawn a new thread. Returns 0 on success, -1 on failure. */
int fuse_rt_thread_spawn(fuse_rt_thread_fn fn, void *arg);

/* ===== Synchronization (Phase 03) ===== */

/* Opaque mutex handle. */
typedef struct fuse_rt_mutex fuse_rt_mutex;

fuse_rt_mutex *fuse_rt_mutex_create(void);
void           fuse_rt_mutex_lock(fuse_rt_mutex *m);
void           fuse_rt_mutex_unlock(fuse_rt_mutex *m);
void           fuse_rt_mutex_destroy(fuse_rt_mutex *m);

/* Opaque condition variable handle. */
typedef struct fuse_rt_cond fuse_rt_cond;

fuse_rt_cond *fuse_rt_cond_create(void);
void          fuse_rt_cond_wait(fuse_rt_cond *c, fuse_rt_mutex *m);
void          fuse_rt_cond_signal(fuse_rt_cond *c);
void          fuse_rt_cond_broadcast(fuse_rt_cond *c);
void          fuse_rt_cond_destroy(fuse_rt_cond *c);

/* ===== Runtime initialization ===== */

/* Initialize the runtime. Called once before main. */
void fuse_rt_init(int argc, char **argv);

#ifdef __cplusplus
}
#endif

#endif /* FUSE_RT_H */
