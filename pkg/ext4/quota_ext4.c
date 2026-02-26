#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <sys/mount.h>
#include <mntent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/sysmacros.h>
#include <linux/quota.h>
#include <sys/quota.h>
#include <limits.h>
#include <sys/utsname.h>
#include "quota_ext4.h"

static char device_path[PATH_MAX] = {0};
static int kernel_version_checked = 0;
static int kernel_supports_getnextquota = 0;

static int check_kernel_version(void) {
    if (kernel_version_checked) {
        return kernel_supports_getnextquota;
    }

    struct utsname uts;
    if (uname(&uts) != 0) {
        kernel_version_checked = 1;
        kernel_supports_getnextquota = 0;
        return 0;
    }

    int major, minor, patch;
    if (sscanf(uts.release, "%d.%d.%d", &major, &minor, &patch) != 3) {
        kernel_version_checked = 1;
        kernel_supports_getnextquota = 0;
        return 0;
    }

    kernel_version_checked = 1;
    if (major > 4 || (major == 4 && minor >= 6)) {
        kernel_supports_getnextquota = 1;
    } else {
        kernel_supports_getnextquota = 0;
    }

    return kernel_supports_getnextquota;
}

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
            char fstype[64];
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
    
    if (best_match) {
        return find_device_by_major_minor(best_major, best_minor);
    }
    
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
            dq->dqb_ihardlimit > 0 || dq->dqb_isoftlimit > 0 ||
            dq->dqb_curspace > 0 || dq->dqb_curinodes > 0);
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

    int use_nextquota = 0;
    if (check_kernel_version()) {
        struct if_nextdqblk test_dq;
        memset(&test_dq, 0, sizeof(test_dq));
        int test_ret = quotactl(QCMD(Q_GETNEXTQUOTA, type), device_path, 0, (caddr_t)&test_dq);
        if (test_ret >= 0) {
            use_nextquota = 1;
        }
    }

    if (use_nextquota) {
        uint32_t next_id = 0;
        while (next_id <= (uint32_t)max_id && count < 100000) {
            struct if_nextdqblk dq;
            memset(&dq, 0, sizeof(dq));

            int ret = quotactl(QCMD(Q_GETNEXTQUOTA, type), device_path, next_id, (caddr_t)&dq);

            if (ret < 0) {
                break;
            }

            if (!has_quota_set((struct if_dqblk *)&dq)) {
                next_id = dq.dqb_id + 1;
                continue;
            }

            EXT4QuotaInfo info;
            info.id = dq.dqb_id;
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
            next_id = dq.dqb_id + 1;
        }
    } else {
        uint32_t scan_limit = (max_id > 0) ? (uint32_t)max_id : 65536;
        uint32_t step = 1;
        
        if (scan_limit > 10000000) {
            step = 100000;
        } else if (scan_limit > 1000000) {
            step = 10000;
        } else if (scan_limit > 100000) {
            step = 1000;
        } else if (scan_limit > 10000) {
            step = 100;
        } else if (scan_limit > 1000) {
            step = 10;
        }
        
        int consecutive_errors = 0;
        const int max_consecutive_errors = 1000;
        
        for (uint32_t id = 0; id <= scan_limit; id += step) {
            struct if_dqblk dq;
            memset(&dq, 0, sizeof(dq));

            int ret = quotactl(QCMD(Q_GETQUOTA, type), device_path, id, (caddr_t)&dq);

            if (ret < 0) {
                consecutive_errors++;
                if (consecutive_errors >= max_consecutive_errors) {
                    break;
                }
                continue;
            }

            consecutive_errors = 0;

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
            
            if (step > 1) {
                for (uint32_t check_id = id + 1; check_id < id + step && check_id <= scan_limit; check_id++) {
                    struct if_dqblk check_dq;
                    memset(&check_dq, 0, sizeof(check_dq));
                    int check_ret = quotactl(QCMD(Q_GETQUOTA, type), device_path, check_id, (caddr_t)&check_dq);
                    if (check_ret >= 0 && has_quota_set(&check_dq)) {
                        EXT4QuotaInfo check_info;
                        check_info.id = check_id;
                        check_info.qtype = type;
                        check_info.bhardlimit = check_dq.dqb_bhardlimit;
                        check_info.bsoftlimit = check_dq.dqb_bsoftlimit;
                        check_info.curblocks = check_dq.dqb_curspace / 1024;
                        check_info.ihardlimit = check_dq.dqb_ihardlimit;
                        check_info.isoftlimit = check_dq.dqb_isoftlimit;
                        check_info.curinodes = check_dq.dqb_curinodes;
                        check_info.btime = check_dq.dqb_btime;
                        check_info.itime = check_dq.dqb_itime;

                        if (count >= capacity) {
                            capacity *= 2;
                            EXT4QuotaInfo *new_items = (EXT4QuotaInfo *)realloc(items, capacity * sizeof(EXT4QuotaInfo));
                            if (!new_items) {
                                free(items);
                                return ENOMEM;
                            }
                            items = new_items;
                        }
                        items[count++] = check_info;
                    }
                }
            }
        }
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
