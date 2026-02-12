#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <sys/mount.h>
#include <mntent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <linux/quota.h>
#include <sys/quota.h>
#include <limits.h>
#include "quota_ext4.h"

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

const char* ext4_error_string(int error_code) {
    switch (error_code) {
        case 0:
            return "Success";
        case EINVAL:
            return "Invalid argument";
        case ENOENT:
            return "No such file or directory";
        case ENODEV:
            return "No such device";
        case EPERM:
            return "Operation not permitted";
        case EACCES:
            return "Permission denied";
        case ESRCH:
            return "No such process";
        case ENOSPC:
            return "No space left on device";
        case EBUSY:
            return "Device or resource busy";
        case EEXIST:
            return "File exists";
        case ENOTDIR:
            return "Not a directory";
        case EISDIR:
            return "Is a directory";
        default:
            return strerror(error_code);
    }
}

int ext4_set_quota(const char *path, uint32_t id, int type,
                    uint64_t bhard, uint64_t bsoft,
                    uint64_t ihard, uint64_t isoft) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct if_dqblk dq;
    memset(&dq, 0, sizeof(dq));

    dq.dqb_bhardlimit = bhard;
    dq.dqb_bsoftlimit = bsoft;
    dq.dqb_ihardlimit = ihard;
    dq.dqb_isoftlimit = isoft;
    dq.dqb_valid = QIF_BLIMITS | QIF_ILIMITS;

    int ret = quotactl(QCMD(Q_SETQUOTA, type), device_path, id, (caddr_t)&dq);

    if (ret < 0) {
        return errno;
    }

    return 0;
}

int ext4_get_quota(const char *path, uint32_t id, int type, EXT4QuotaInfo *info) {
    if (!path || !info) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct if_dqblk dq;
    memset(&dq, 0, sizeof(dq));

    int ret = quotactl(QCMD(Q_GETQUOTA, type), device_path, id, (caddr_t)&dq);

    if (ret < 0) {
        return errno;
    }

    info->id = id;
    info->qtype = type;
    info->bhardlimit = dq.dqb_bhardlimit;
    info->bsoftlimit = dq.dqb_bsoftlimit;
    info->curblocks = dq.dqb_curspace / 1024;
    info->ihardlimit = dq.dqb_ihardlimit;
    info->isoftlimit = dq.dqb_isoftlimit;
    info->curinodes = dq.dqb_curinodes;
    info->btime = dq.dqb_btime;
    info->itime = dq.dqb_itime;

    return 0;
}

static int has_quota_set(const struct if_dqblk *dq) {
    return (dq->dqb_bhardlimit > 0 || dq->dqb_bsoftlimit > 0 ||
            dq->dqb_ihardlimit > 0 || dq->dqb_isoftlimit > 0);
}

int ext4_list_quotas(const char *path, int type, EXT4QuotaList *list, int max_id) {
    if (!path || !list) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    EXT4QuotaInfo *items = NULL;
    size_t count = 0;
    size_t capacity = 1024;

    items = (EXT4QuotaInfo *)malloc(capacity * sizeof(EXT4QuotaInfo));
    if (!items) {
        return ENOMEM;
    }

    for (uint32_t id = 0; id <= (uint32_t)max_id; id++) {
        struct if_dqblk dq;
        memset(&dq, 0, sizeof(dq));

        int ret = quotactl(QCMD(Q_GETQUOTA, type), device_path, id, (caddr_t)&dq);

        if (ret < 0) {
            continue;
        }

        if (!has_quota_set(&dq)) {
        continue;
    }

    EXT4QuotaInfo info;
    info.id = id;
    info.qtype = type;
    info.bhardlimit = dq.dqb_bhardlimit;
    info.bsoftlimit = dq.dqb_bsoftlimit;
    info.curblocks = dq.dqb_curspace / 1024;
    info.ihardlimit = dq.dqb_ihardlimit;
    info.isoftlimit = dq.dqb_isoftlimit;
    info.curinodes = dq.dqb_curinodes;
    info.btime = dq.dqb_btime;
    info.itime = dq.dqb_itime;

    if (count >= capacity) {
        capacity *= 2;
        EXT4QuotaInfo *new_items = (EXT4QuotaInfo *)realloc(items, capacity * sizeof(EXT4QuotaInfo));
        if (!new_items) {
            free(items);
            return ENOMEM;
        }
        items = new_items;
    }
    items[count++] = info;
    }

    list->items = items;
    list->count = count;

    return 0;
}

void ext4_free_quota_list(EXT4QuotaList *list) {
    if (list && list->items) {
        free(list->items);
        list->items = NULL;
        list->count = 0;
    }
}

int ext4_remove_quota(const char *path, uint32_t id, int type) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct if_dqblk dq;
    memset(&dq, 0, sizeof(dq));

    dq.dqb_valid = QIF_BLIMITS | QIF_ILIMITS;

    int ret = quotactl(QCMD(Q_SETQUOTA, type), device_path, id, (caddr_t)&dq);

    if (ret < 0) {
        return errno;
    }

    return 0;
}

int ext4_test_quota(const char *path, uint32_t id, int type) {
    if (!path) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    struct if_dqblk dq;
    memset(&dq, 0, sizeof(dq));

    int ret = quotactl(QCMD(Q_GETQUOTA, type), device_path, id, (caddr_t)&dq);

    if (ret < 0) {
        return errno;
    }

    if (dq.dqb_bhardlimit == 0 && dq.dqb_bsoftlimit == 0 &&
        dq.dqb_ihardlimit == 0 && dq.dqb_isoftlimit == 0) {
        return ENOENT;
    }

    return 0;
}
