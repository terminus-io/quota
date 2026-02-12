#ifndef QUOTA_XFS_H
#define QUOTA_XFS_H

#include <stdint.h>

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
} XFSQuotaInfo;

typedef struct {
    XFSQuotaInfo *items;
    int count;
    int capacity;
} XFSQuotaList;

int xfs_set_quota(const char *path, uint32_t id, int type, 
                  uint64_t bhard, uint64_t bsoft, 
                  uint64_t ihard, uint64_t isoft);

int xfs_get_quota(const char *path, uint32_t id, int type, XFSQuotaInfo *info);

int xfs_list_quotas(const char *path, int type, XFSQuotaList *list, int max_id);

void xfs_free_quota_list(XFSQuotaList *list);

int xfs_test_quota(const char *path, uint32_t id, int type);

int xfs_remove_quota(const char *path, uint32_t id, int type);

const char* xfs_error_string(int err);

#ifdef __cplusplus
}
#endif

#endif
