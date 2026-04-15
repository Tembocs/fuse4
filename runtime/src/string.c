#include "fuse_rt.h"
#include <string.h>

void fuse_rt_string_concat(const uint8_t *a, size_t a_len,
                            const uint8_t *b, size_t b_len,
                            uint8_t **out_data, size_t *out_len) {
    size_t total = a_len + b_len;
    uint8_t *data = (uint8_t *)fuse_rt_mem_alloc(total);
    if (a_len > 0) {
        memcpy(data, a, a_len);
    }
    if (b_len > 0) {
        memcpy(data + a_len, b, b_len);
    }
    *out_data = data;
    *out_len = total;
}
