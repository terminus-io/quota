#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <sys/quota.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/sysmacros.h>
#include <linux/dqblk_xfs.h>
#include <mntent.h>
#include <limits.h>
#include "quota_xfs.h"

static char device_path[PATH_MAX] = {0};

static int find_device_for_path(const char *path) {
    FILE *fp = setmntent("/proc/mounts", "r");
    if (!fp) {
        return -1;
    }

    struct mntent *ent;
    while ((ent = getmntent(fp)) != NULL) {
        if (strcmp(ent->mnt_dir, path) == 0) {
            strncpy(device_path, ent->mnt_fsname, PATH_MAX - 1);
            device_path[PATH_MAX - 1] = '\0';
            endmntent(fp);
            return 0;
        }
    }

    endmntent(fp);
    return -1;
}

#define XFS_QUOTA_USRQUOTA 0
#define XFS_QUOTA_GRPQUOTA 1
#define XFS_QUOTA_PRJQUOTA 2

static char error_buffer[256];

const char* xfs_error_string(int err) {
    if (err == 0) {
        return "Success";
    }
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-result"
    strerror_r(err, error_buffer, sizeof(error_buffer));
#pragma GCC diagnostic pop
    return error_buffer;
}

int xfs_set_quota(const char *path, uint32_t id, int type, 
                  uint64_t bhard, uint64_t bsoft, 
                  uint64_t ihard, uint64_t isoft) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct fs_disk_quota dq;
    memset(&dq, 0, sizeof(dq));

    dq.d_version = FS_DQUOT_VERSION;
    dq.d_id = id;
    dq.d_flags = type;

    if (bhard > 0) {
        dq.d_blk_hardlimit = bhard * 2;
    }
    if (bsoft > 0) {
        dq.d_blk_softlimit = bsoft * 2;
    }
    if (ihard > 0) {
        dq.d_ino_hardlimit = ihard;
    }
    if (isoft > 0) {
        dq.d_ino_softlimit = isoft;
    }

    dq.d_fieldmask = FS_DQ_LIMIT_MASK;

    int ret = quotactl(QCMD(Q_XSETQLIM, type), device_path, id, (caddr_t)&dq);
    if (ret < 0) {
        return errno;
    }

    return 0;
}

int xfs_get_quota(const char *path, uint32_t id, int type, XFSQuotaInfo *info) {
    if (!path || !info) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct fs_disk_quota dq;
    memset(&dq, 0, sizeof(dq));

    int ret = quotactl(QCMD(Q_XGETQUOTA, type), device_path, id, (caddr_t)&dq);
    
    if (ret < 0) {
        return errno;
    }

    info->id = id;
    info->qtype = type;
    info->bhardlimit = dq.d_blk_hardlimit / 2;
    info->bsoftlimit = dq.d_blk_softlimit / 2;
    info->curblocks = dq.d_bcount / 2;
    info->ihardlimit = dq.d_ino_hardlimit;
    info->isoftlimit = dq.d_ino_softlimit;
    info->curinodes = dq.d_icount;
    info->btime = dq.d_btimer;
    info->itime = dq.d_itimer;

    return 0;
}

static int has_quota_set(const struct fs_disk_quota *dq) {
    return (dq->d_blk_hardlimit > 0 || dq->d_blk_softlimit > 0 ||
            dq->d_ino_hardlimit > 0 || dq->d_ino_softlimit > 0);
}

int xfs_list_quotas(const char *path, int type, XFSQuotaList *list, int max_id) {
    if (!path || !list) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    list->count = 0;
    list->capacity = max_id > 0 ? max_id : 65536;
    list->items = (XFSQuotaInfo *)calloc(list->capacity, sizeof(XFSQuotaInfo));
    
    if (!list->items) {
        return ENOMEM;
    }

    int found = 0;
    
    for (uint32_t id = 0; id < (uint32_t)list->capacity && found < list->capacity; id++) {
        struct fs_disk_quota dq;
        memset(&dq, 0, sizeof(dq));

        int ret = quotactl(QCMD(Q_XGETQUOTA, type), device_path, id, (caddr_t)&dq);
        
        if (ret < 0) {
            continue;
        }

        if (!has_quota_set(&dq)) {
            continue;
        }

        XFSQuotaInfo *info = &list->items[found];
        info->id = id;
        info->qtype = type;
        info->bhardlimit = dq.d_blk_hardlimit / 2;
        info->bsoftlimit = dq.d_blk_softlimit / 2;
        info->curblocks = dq.d_bcount / 2;
        info->ihardlimit = dq.d_ino_hardlimit;
        info->isoftlimit = dq.d_ino_softlimit;
        info->curinodes = dq.d_icount;
        info->btime = dq.d_btimer;
        info->itime = dq.d_itimer;
        
        found++;
    }

    list->count = found;
    return 0;
}

void xfs_free_quota_list(XFSQuotaList *list) {
    if (list && list->items) {
        free(list->items);
        list->items = NULL;
        list->count = 0;
        list->capacity = 0;
    }
}

int xfs_test_quota(const char *path, uint32_t id, int type) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct fs_disk_quota dq;
    memset(&dq, 0, sizeof(dq));

    int ret = quotactl(QCMD(Q_XGETQUOTA, type), device_path, id, (caddr_t)&dq);
    
    if (ret < 0) {
        return errno;
    }

    return 0;
}

int xfs_remove_quota(const char *path, uint32_t id, int type) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct fs_disk_quota dq;
    memset(&dq, 0, sizeof(dq));
    
    dq.d_version = FS_DQUOT_VERSION;
    dq.d_id = id;
    dq.d_flags = type;
    dq.d_fieldmask = FS_DQ_LIMIT_MASK;

    int ret = quotactl(QCMD(Q_XSETQLIM, type), device_path, id, (caddr_t)&dq);
    if (ret < 0) {
        return errno;
    }

    return 0;
}
