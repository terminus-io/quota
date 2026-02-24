#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <mntent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <linux/quota.h>
#include <limits.h>
#include "quota_ext4.h"

#define QUOTA_VERSION "2.1"
#define V2_DQBLK_SIZE 148

static char quota_file_path[PATH_MAX] = {0};

static int find_quota_file(const char *path, int type) {
    FILE *fp = setmntent("/proc/mounts", "r");
    if (!fp) {
        return -1;
    }

    const char *quota_file = NULL;
    if (type == USRQUOTA) {
        quota_file = "aquota.user";
    } else if (type == GRPQUOTA) {
        quota_file = "aquota.group";
    } else if (type == PRJQUOTA) {
        quota_file = "aquota.project";
    }

    if (!quota_file) {
        endmntent(fp);
        return -1;
    }

    struct mntent *ent;
    while ((ent = getmntent(fp)) != NULL) {
        if (strcmp(ent->mnt_dir, path) == 0) {
            endmntent(fp);
            snprintf(quota_file_path, PATH_MAX, "%s/%s", path, quota_file);
            return 0;
        }
    }

    endmntent(fp);

    snprintf(quota_file_path, PATH_MAX, "%s/%s", path, quota_file);
    return 0;
}

struct v2_disk_dqinfo {
    __u32 dqi_bgrace;
    __u32 dqi_igrace;
    __u32 dqi_flags;
    __u32 dqi_blocks;
    __u32 dqi_free_blk;
    __u32 dqi_free_entry;
};

struct v2_disk_dqblk {
    __u32 dqb_id;
    __u32 dqb_ihardlimit;
    __u32 dqb_isoftlimit;
    __u32 dqb_curinodes;
    __u32 dqb_bhardlimit;
    __u32 dqb_bsoftlimit;
    __u64 dqb_curspace;
    __u64 dqb_btime;
    __u64 dqb_itime;
};

static int read_quota_file(const char *path, int type, EXT4QuotaList *list) {
    if (find_quota_file(path, type) != 0) {
        return ENOENT;
    }

    int fd = open(quota_file_path, O_RDONLY);
    if (fd < 0) {
        return errno;
    }

    struct v2_disk_dqinfo dqinfo;
    if (read(fd, &dqinfo, sizeof(dqinfo)) != sizeof(dqinfo)) {
        close(fd);
        return EIO;
    }

    list->count = 0;
    list->capacity = 1024;
    list->items = (EXT4QuotaInfo *)malloc(list->capacity * sizeof(EXT4QuotaInfo));
    if (!list->items) {
        close(fd);
        return ENOMEM;
    }

#define BUFFER_SIZE 4096
    struct v2_disk_dqblk buffer[BUFFER_SIZE];
    ssize_t bytes_read;
    int i;

    while ((bytes_read = read(fd, buffer, sizeof(buffer))) > 0) {
        int count = bytes_read / sizeof(struct v2_disk_dqblk);
        
        for (i = 0; i < count; i++) {
            if (buffer[i].dqb_id == 0 || buffer[i].dqb_id == ~0U) {
                continue;
            }

            if (list->count >= list->capacity) {
                list->capacity *= 2;
                EXT4QuotaInfo *new_items = (EXT4QuotaInfo *)realloc(list->items, list->capacity * sizeof(EXT4QuotaInfo));
                if (!new_items) {
                    free(list->items);
                    list->items = NULL;
                    close(fd);
                    return ENOMEM;
                }
                list->items = new_items;
            }

            EXT4QuotaInfo *info = &list->items[list->count];
            info->id = buffer[i].dqb_id;
            info->qtype = type;
            info->bhardlimit = buffer[i].dqb_bhardlimit;
            info->bsoftlimit = buffer[i].dqb_bsoftlimit;
            info->curblocks = buffer[i].dqb_curspace / 1024;
            info->ihardlimit = buffer[i].dqb_ihardlimit;
            info->isoftlimit = buffer[i].dqb_isoftlimit;
            info->curinodes = buffer[i].dqb_curinodes;
            info->btime = buffer[i].dqb_btime;
            info->itime = buffer[i].dqb_itime;
            
            list->count++;
        }
    }

    close(fd);
    return 0;
}

int ext4_list_quotas_direct(const char *path, int type, EXT4QuotaList *list, int max_id) {
    if (!path || !list) {
        return EINVAL;
    }

    list->count = 0;
    list->capacity = 1024;
    list->items = (EXT4QuotaInfo *)malloc(list->capacity * sizeof(EXT4QuotaInfo));
    if (!list->items) {
        return ENOMEM;
    }

    int ret = read_quota_file(path, type, list);
    if (ret != 0) {
        free(list->items);
        list->items = NULL;
        return ret;
    }

    return 0;
}

int ext4_list_quotas_direct_debug(const char *path, int type, EXT4QuotaList *list, int max_id, char *error_msg, size_t error_msg_size) {
    if (!path || !list) {
        snprintf(error_msg, error_msg_size, "Invalid arguments");
        return EINVAL;
    }

    list->count = 0;
    list->capacity = 1024;
    list->items = (EXT4QuotaInfo *)malloc(list->capacity * sizeof(EXT4QuotaInfo));
    if (!list->items) {
        snprintf(error_msg, error_msg_size, "Memory allocation failed");
        return ENOMEM;
    }

    if (find_quota_file(path, type) != 0) {
        snprintf(error_msg, error_msg_size, "find_quota_file failed for path=%s, type=%d", path, type);
        free(list->items);
        list->items = NULL;
        return ENOENT;
    }

    snprintf(error_msg, error_msg_size, "quota_file_path=%s", quota_file_path);

    int fd = open(quota_file_path, O_RDONLY);
    if (fd < 0) {
        snprintf(error_msg, error_msg_size, "open failed for %s: %s", quota_file_path, strerror(errno));
        free(list->items);
        list->items = NULL;
        return errno;
    }

    struct v2_disk_dqinfo dqinfo;
    if (read(fd, &dqinfo, sizeof(dqinfo)) != sizeof(dqinfo)) {
        snprintf(error_msg, error_msg_size, "read dqinfo failed");
        close(fd);
        free(list->items);
        list->items = NULL;
        return EIO;
    }

    close(fd);

    int ret = read_quota_file(path, type, list);
    if (ret != 0) {
        snprintf(error_msg, error_msg_size, "read_quota_file failed: %s", strerror(ret));
        free(list->items);
        list->items = NULL;
        return ret;
    }

    return 0;
}
