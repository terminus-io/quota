#ifndef QUOTA_EXT4_H
#define QUOTA_EXT4_H

#include <stdint.h>
#include <linux/quota.h>
#include <sys/quota.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    uint32_t id;
    int qtype;
    uint64_t bhardlimit;
    uint64_t bsoftlimit;
    uint64_t curblocks;
    uint64_t ihardlimit;
    uint64_t isoftlimit;
    uint64_t curinodes;
    uint64_t btime;
    uint64_t itime;
} EXT4QuotaInfo;

typedef struct {
    EXT4QuotaInfo *items;
    size_t count;
} EXT4QuotaList;

int ext4_set_quota(const char *path, uint32_t id, int type,
                    uint64_t bhard, uint64_t bsoft,
                    uint64_t ihard, uint64_t isoft);

int ext4_get_quota(const char *path, uint32_t id, int type, EXT4QuotaInfo *info);

int ext4_list_quotas(const char *path, int type, EXT4QuotaList *list, int max_id);

void ext4_free_quota_list(EXT4QuotaList *list);

int ext4_remove_quota(const char *path, uint32_t id, int type);

int ext4_test_quota(const char *path, uint32_t id, int type);

const char* ext4_error_string(int error_code);

#ifdef __cplusplus
}
#endif

#endif
