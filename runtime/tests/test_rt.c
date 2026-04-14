/*
 * test_rt.c — Runtime test suite.
 *
 * Compile and run: see runtime Makefile target.
 * Each test prints PASS/FAIL. Returns non-zero on any failure.
 */

#include "fuse_rt.h"
#include <stdio.h>
#include <string.h>

static int failures = 0;

#define TEST(name) static void name(void)
#define ASSERT(cond, msg) do { \
    if (!(cond)) { \
        fprintf(stderr, "FAIL: %s: %s\n", __func__, msg); \
        failures++; \
        return; \
    } \
} while (0)
#define PASS() fprintf(stderr, "PASS: %s\n", __func__)

/* ===== Memory tests ===== */

TEST(test_mem_alloc_free) {
    void *p = fuse_rt_mem_alloc(128);
    ASSERT(p != NULL, "alloc returned NULL");
    memset(p, 0xAB, 128);
    fuse_rt_mem_free(p);
    PASS();
}

TEST(test_mem_alloc_zeroed) {
    unsigned char *p = (unsigned char *)fuse_rt_mem_alloc_zeroed(16, 1);
    ASSERT(p != NULL, "calloc returned NULL");
    int all_zero = 1;
    for (int i = 0; i < 16; i++) {
        if (p[i] != 0) { all_zero = 0; break; }
    }
    ASSERT(all_zero, "memory not zeroed");
    fuse_rt_mem_free(p);
    PASS();
}

TEST(test_mem_realloc) {
    void *p = fuse_rt_mem_alloc(16);
    ASSERT(p != NULL, "alloc returned NULL");
    p = fuse_rt_mem_realloc(p, 256);
    ASSERT(p != NULL, "realloc returned NULL");
    fuse_rt_mem_free(p);
    PASS();
}

/* ===== IO tests ===== */

TEST(test_io_write_stdout) {
    const char *msg = "runtime test output\n";
    int64_t n = fuse_rt_io_write_stdout(msg, strlen(msg));
    ASSERT(n > 0, "write_stdout returned <= 0");
    PASS();
}

TEST(test_io_write_stderr) {
    const char *msg = "";  /* empty write */
    int64_t n = fuse_rt_io_write_stderr(msg, 0);
    ASSERT(n >= 0, "write_stderr returned < 0");
    PASS();
}

TEST(test_io_file_roundtrip) {
    const char *path = "_fuse_rt_test_tmp.txt";
    const char *data = "hello fuse runtime";

    /* Write */
    int64_t wh = fuse_rt_io_file_open(path, 1);
    ASSERT(wh >= 0, "file_open(write) failed");
    int64_t written = fuse_rt_io_file_write(wh, data, strlen(data));
    ASSERT(written == (int64_t)strlen(data), "file_write count mismatch");
    fuse_rt_io_file_close(wh);

    /* Read back */
    int64_t rh = fuse_rt_io_file_open(path, 0);
    ASSERT(rh >= 0, "file_open(read) failed");
    char buf[64] = {0};
    int64_t nread = fuse_rt_io_file_read(rh, buf, sizeof(buf));
    ASSERT(nread == (int64_t)strlen(data), "file_read count mismatch");
    ASSERT(memcmp(buf, data, strlen(data)) == 0, "data mismatch");
    fuse_rt_io_file_close(rh);

    /* Cleanup */
    remove(path);
    PASS();
}

/* ===== Process tests ===== */

TEST(test_proc_argc) {
    int argc = fuse_rt_proc_argc();
    ASSERT(argc >= 1, "argc should be >= 1");
    PASS();
}

TEST(test_proc_argv) {
    const char *arg0 = fuse_rt_proc_argv(0);
    ASSERT(arg0 != NULL, "argv[0] should not be NULL");
    PASS();
}

TEST(test_proc_env) {
    /* PATH should exist on all platforms. */
    const char *path = fuse_rt_proc_env_get("PATH");
    ASSERT(path != NULL, "PATH env should exist");
    PASS();
}

/* ===== Time tests ===== */

TEST(test_time_monotonic) {
    int64_t t1 = fuse_rt_time_now_ns();
    int64_t t2 = fuse_rt_time_now_ns();
    ASSERT(t1 > 0, "time should be positive");
    ASSERT(t2 >= t1, "time should be monotonic");
    PASS();
}

/* ===== Sync tests ===== */

TEST(test_mutex_basic) {
    fuse_rt_mutex *m = fuse_rt_mutex_create();
    ASSERT(m != NULL, "mutex_create returned NULL");
    fuse_rt_mutex_lock(m);
    fuse_rt_mutex_unlock(m);
    fuse_rt_mutex_destroy(m);
    PASS();
}

TEST(test_cond_basic) {
    fuse_rt_cond *c = fuse_rt_cond_create();
    ASSERT(c != NULL, "cond_create returned NULL");
    /* Just test create/destroy without waiting. */
    fuse_rt_cond_destroy(c);
    PASS();
}

/* ===== Thread tests ===== */

static volatile int thread_flag = 0;

static void thread_fn(void *arg) {
    (void)arg;
    thread_flag = 1;
}

TEST(test_thread_spawn) {
    thread_flag = 0;
    int err = fuse_rt_thread_spawn(thread_fn, NULL);
    ASSERT(err == 0, "thread_spawn failed");
    /* Give the thread a moment to run. */
    for (int i = 0; i < 1000000 && !thread_flag; i++) {
        /* spin */
    }
    ASSERT(thread_flag == 1, "thread did not run");
    PASS();
}

/* ===== Main ===== */

int main(int argc, char **argv) {
    fuse_rt_init(argc, argv);

    test_mem_alloc_free();
    test_mem_alloc_zeroed();
    test_mem_realloc();
    test_io_write_stdout();
    test_io_write_stderr();
    test_io_file_roundtrip();
    test_proc_argc();
    test_proc_argv();
    test_proc_env();
    test_time_monotonic();
    test_mutex_basic();
    test_cond_basic();
    test_thread_spawn();

    if (failures > 0) {
        fprintf(stderr, "\n%d test(s) FAILED\n", failures);
        return 1;
    }
    fprintf(stderr, "\nAll runtime tests passed.\n");
    return 0;
}
