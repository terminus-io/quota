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
#include <sys/utsname.h>
#include <mntent.h>
#include <limits.h>
#include "quota_xfs.h"

static char device_path[PATH_MAX] = {0};

static int find_device_by_major_minor(unsigned int major, unsigned int minor) {
    char uevent_path[PATH_MAX];
    snprintf(uevent_path, PATH_MAX, "/sys/dev/block/%u:%u/uevent", major, minor);
    
    FILE *fp = fopen(uevent_path, "r");
    if (!fp) {
        return -1;
    }
    
    char line[PATH_MAX];
    char devname[PATH_MAX] = {0};
    
    while (fgets(line, sizeof(line), fp)) {
        if (strncmp(line, "DEVNAME=", 8) == 0) {
            strncpy(devname, line + 8, PATH_MAX - 1);
            size_t len = strlen(devname);
            if (len > 0 && devname[len - 1] == '\n') {
                devname[len - 1] = '\0';
            }
            break;
        }
    }
    fclose(fp);
    
    if (devname[0] == '\0') {
        return -1;
    }
    
    const char *dev_paths[] = {
        "/dev/mapper/%s",
        "/dev/%s",
        "/dev/block/%s",
        "/dev/disk/by-uuid/%s",
        "/dev/disk/by-label/%s",
        "/tmp/quota_%s",
        NULL
    };
    
    for (int i = 0; dev_paths[i] != NULL; i++) {
        char test_path[PATH_MAX];
        snprintf(test_path, PATH_MAX, dev_paths[i], devname);
        
        if (access(test_path, F_OK) == 0) {
            strncpy(device_path, test_path, PATH_MAX - 1);
            device_path[PATH_MAX - 1] = '\0';
            return 0;
        }
    }
    
    char fake_device_path[PATH_MAX];
    snprintf(fake_device_path, PATH_MAX, "/tmp/quota_%u_%u", major, minor);
    
    if (access(fake_device_path, F_OK) == 0) {
        strncpy(device_path, fake_device_path, PATH_MAX - 1);
        device_path[PATH_MAX - 1] = '\0';
        return 0;
    }
    
    if (mknod(fake_device_path, S_IFBLK | 0600, makedev(major, minor)) == 0) {
        strncpy(device_path, fake_device_path, PATH_MAX - 1);
        device_path[PATH_MAX - 1] = '\0';
        return 0;
    }
    
    return -1;
}

static int is_valid_mount_point(const char *mount_point, const char *path) {
    struct stat st_mount, st_path;
    
    if (stat(mount_point, &st_mount) != 0) {
        return 0;
    }
    
    if (stat(path, &st_path) != 0) {
        return 0;
    }
    
    if (st_mount.st_dev != st_path.st_dev) {
        return 0;
    }
    
    return 1;
}

static int find_device_for_path(const char *path) {
    FILE *fp = fopen("/proc/self/mountinfo", "r");
    if (!fp) {
        return -1;
    }
    
    char line[PATH_MAX * 4];
    size_t path_len = strlen(path);
    char *best_match = NULL;
    unsigned int best_major = 0;
    unsigned int best_minor = 0;
    size_t best_match_len = 0;
    char best_fstype[64] = {0};
    int is_root_path = (path_len == 1 && path[0] == '/');
    
    while (fgets(line, sizeof(line), fp)) {
        char mount_point[PATH_MAX];
        char root[PATH_MAX];
        char fstype[64];
        unsigned int major, minor;
        
        int parsed = sscanf(line, "%*d %*d %u:%u %s %s", &major, &minor, root, mount_point);
        if (parsed != 4) {
            continue;
        }
        
        size_t mnt_len = strlen(mount_point);
        
        if (is_root_path) {
            if (strcmp(mount_point, "/") != 0) {
                continue;
            }
            
            char *fstype_ptr = strstr(line, " - ");
            if (!fstype_ptr) {
                continue;
            }
            
            fstype_ptr += 3;
            sscanf(fstype_ptr, "%63s", fstype);
            
            if (strcmp(fstype, "ext4") == 0 || strcmp(fstype, "xfs") == 0) {
                if (is_valid_mount_point(mount_point, path)) {
                    best_match = mount_point;
                    best_major = major;
                    best_minor = minor;
                    best_match_len = mnt_len;
                    strncpy(best_fstype, fstype, sizeof(best_fstype) - 1);
                    best_fstype[sizeof(best_fstype) - 1] = '\0';
                    break;
                }
            }
        } else {
            if (mnt_len > path_len) {
                continue;
            }
            
            if (strncmp(path, mount_point, mnt_len) == 0) {
                if (mnt_len > best_match_len) {
                    if (is_valid_mount_point(mount_point, path)) {
                        best_match = mount_point;
                        best_major = major;
                        best_minor = minor;
                        best_match_len = mnt_len;
                        
                        char *fstype_ptr = strstr(line, " - ");
                        if (fstype_ptr) {
                            fstype_ptr += 3;
                            sscanf(fstype_ptr, "%63s", best_fstype);
                        }
                    }
                }
            }
        }
    }
    
    fclose(fp);
    
    if (!best_match) {
        return -1;
    }
    
    int ret = find_device_by_major_minor(best_major, best_minor);
    return ret;
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
            dq->d_ino_hardlimit > 0 || dq->d_ino_softlimit > 0 ||
            dq->d_bcount > 0 || dq->d_icount > 0);
}

int xfs_list_quotas(const char *path, int type, XFSQuotaList *list, int max_id) {
    (void)max_id;
    
    if (!path || !list) {
        return EINVAL;
    }

    if (find_device_for_path(path) != 0) {
        return ENODEV;
    }

    list->count = 0;
    list->capacity = 1024;
    list->items = (XFSQuotaInfo *)calloc(list->capacity, sizeof(XFSQuotaInfo));
    
    if (!list->items) {
        return ENOMEM;
    }

    int found = 0;
    uint32_t next_id = 0;
    
    while (found < 100000) {
        struct fs_disk_quota dq;
        memset(&dq, 0, sizeof(dq));

        int ret = quotactl(QCMD(Q_XGETNEXTQUOTA, type), device_path, next_id, (caddr_t)&dq);
        
        if (ret < 0) {
            break;
        }

        if (!has_quota_set(&dq)) {
            next_id = dq.d_id + 1;
            continue;
        }

        if (found >= list->capacity) {
            list->capacity *= 2;
            XFSQuotaInfo *new_items = (XFSQuotaInfo *)realloc(list->items, list->capacity * sizeof(XFSQuotaInfo));
            if (!new_items) {
                free(list->items);
                list->items = NULL;
                return ENOMEM;
            }
            list->items = new_items;
        }

        XFSQuotaInfo *info = &list->items[found];
        info->id = dq.d_id;
        info->qtype = type;
        info->bhardlimit = dq.d_blk_hardlimit / 2;
        info->bsoftlimit = dq.d_blk_softlimit / 2;
        info->curblocks = dq.d_bcount / 2;
        info->ihardlimit = dq.d_ino_hardlimit;
        info->isoftlimit = dq.d_ino_softlimit;
        info->curinodes = dq.d_icount;
        info->btime = dq.d_btimer;
        info->itime = dq.d_itimer;
        
        next_id = dq.d_id + 1;
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
