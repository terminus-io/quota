#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <dirent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <fcntl.h>
#include <mntent.h>
#include <limits.h>
#include "quota_ext4.h"

static char device_path[PATH_MAX] = {0};
static int device_number = -1;

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

static int get_device_number(const char *device_path) {
    FILE *fp = fopen("/proc/partitions", "r");
    if (!fp) {
        return -1;
    }

    char line[1024];
    int major = -1, minor = -1;
    char dev_name[256];

    while (fgets(line, sizeof(line), fp)) {
        if (sscanf(line, "%*d %d %d %255s", &major, &minor, dev_name) == 3) {
            if (strstr(device_path, dev_name)) {
                fclose(fp);
                return (major << 8) | minor;
            }
        }
    }

    fclose(fp);
    return -1;
}

static int try_proc_fs_quota(int type, EXT4QuotaList *list) {
    DIR *dir = opendir("/proc/fs/quota");
    if (!dir) {
        return -1;
    }

    struct dirent *entry;
    while ((entry = readdir(dir)) != NULL) {
        if (entry->d_name[0] == '.') {
            continue;
        }

        char quota_path[PATH_MAX];
        snprintf(quota_path, sizeof(quota_path), "/proc/fs/quota/%s", entry->d_name);

        struct stat st;
        if (stat(quota_path, &st) != 0 || !S_ISDIR(st.st_mode)) {
            continue;
        }

        DIR *quota_dir = opendir(quota_path);
        if (!quota_dir) {
            continue;
        }

        struct dirent *q_entry;
        while ((q_entry = readdir(quota_dir)) != NULL) {
            if (q_entry->d_name[0] == '.') {
                continue;
            }

            char type_path[PATH_MAX];
            snprintf(type_path, sizeof(type_path), "%s/%s", quota_path, q_entry->d_name);

            int entry_type = -1;
            if (strcmp(q_entry->d_name, "usrquota") == 0) {
                entry_type = USRQUOTA;
            } else if (strcmp(q_entry->d_name, "grpquota") == 0) {
                entry_type = GRPQUOTA;
            } else if (strcmp(q_entry->d_name, "prjquota") == 0) {
                entry_type = PRJQUOTA;
            }

            if (entry_type != type) {
                continue;
            }

            DIR *type_dir = opendir(type_path);
            if (!type_dir) {
                continue;
            }

            struct dirent *id_entry;
            while ((id_entry = readdir(type_dir)) != NULL) {
                if (id_entry->d_name[0] == '.') {
                    continue;
                }

                uint32_t id = strtoul(id_entry->d_name, NULL, 10);
                if (id == 0) {
                    continue;
                }

                char id_path[PATH_MAX];
                snprintf(id_path, sizeof(id_path), "%s/%s", type_path, id_entry->d_name);

                FILE *fp = fopen(id_path, "r");
                if (!fp) {
                    continue;
                }

                EXT4QuotaInfo info;
                memset(&info, 0, sizeof(info));
                info.id = id;
                info.qtype = type;

                char line[1024];
                while (fgets(line, sizeof(line), fp)) {
                    if (strncmp(line, "block_hard_limit:", 17) == 0) {
                        sscanf(line + 17, "%lu", &info.bhardlimit);
                    } else if (strncmp(line, "block_soft_limit:", 17) == 0) {
                        sscanf(line + 17, "%lu", &info.bsoftlimit);
                    } else if (strncmp(line, "block_current:", 14) == 0) {
                        unsigned long long cur;
                        sscanf(line + 14, "%llu", &cur);
                        info.curblocks = cur / 1024;
                    } else if (strncmp(line, "inode_hard_limit:", 17) == 0) {
                        sscanf(line + 17, "%lu", &info.ihardlimit);
                    } else if (strncmp(line, "inode_soft_limit:", 17) == 0) {
                        sscanf(line + 17, "%lu", &info.isoftlimit);
                    } else if (strncmp(line, "inode_current:", 14) == 0) {
                        unsigned long long cur;
                        sscanf(line + 14, "%llu", &cur);
                        info.curinodes = cur;
                    }
                }

                fclose(fp);

                if (info.bhardlimit > 0 || info.bsoftlimit > 0 ||
                    info.ihardlimit > 0 || info.isoftlimit > 0) {
                    if (list->count >= list->capacity) {
                        list->capacity *= 2;
                        EXT4QuotaInfo *new_items = (EXT4QuotaInfo *)realloc(list->items, list->capacity * sizeof(EXT4QuotaInfo));
                        if (!new_items) {
                            closedir(type_dir);
                            closedir(quota_dir);
                            closedir(dir);
                            return ENOMEM;
                        }
                        list->items = new_items;
                    }
                    list->items[list->count++] = info;
                }
            }

            closedir(type_dir);
        }

        closedir(quota_dir);
    }

    closedir(dir);
    return 0;
}

int ext4_list_quotas_fast(const char *path, int type, EXT4QuotaList *list, int max_id) {
    if (!path || !list) {
        return EINVAL;
    }

    list->count = 0;
    list->capacity = 1024;
    list->items = (EXT4QuotaInfo *)malloc(list->capacity * sizeof(EXT4QuotaInfo));
    if (!list->items) {
        return ENOMEM;
    }

    int ret = try_proc_fs_quota(type, list);
    if (ret == 0) {
        return 0;
    }

    return ENOTSUP;
}
