package quota

/*
#cgo CFLAGS: -Wall -Wextra
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <linux/fs.h>
#include <linux/fsmap.h>
#include <mntent.h>
#include <limits.h>
#include <stdint.h>
#include "pkg/xfs/quota_xfs.h"

static char error_buffer[256];

const char* project_error_string(int error_code) {
    if (error_code == 0) {
        return "Success";
    }
    strerror_r(error_code, error_buffer, sizeof(error_buffer));
    return error_buffer;
}

int set_project_id_xfs(const char *path, uint32_t project_id) {
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        return errno;
    }

    struct fsxattr attr;
    memset(&attr, 0, sizeof(attr));

    if (ioctl(fd, FS_IOC_FSGETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    attr.fsx_projid = project_id;
    attr.fsx_xflags |= FS_XFLAG_PROJINHERIT;

    if (ioctl(fd, FS_IOC_FSSETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    close(fd);
    return 0;
}

int set_project_id_ext4(const char *path, uint32_t project_id) {
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        return errno;
    }

    struct fsxattr attr;
    memset(&attr, 0, sizeof(attr));

    if (ioctl(fd, FS_IOC_FSGETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    attr.fsx_projid = project_id;
    attr.fsx_xflags |= FS_XFLAG_PROJINHERIT;

    if (ioctl(fd, FS_IOC_FSSETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    close(fd);
    return 0;
}

int get_project_id(const char *path, uint32_t *project_id) {
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        return errno;
    }

    struct fsxattr attr;
    memset(&attr, 0, sizeof(attr));

    if (ioctl(fd, FS_IOC_FSGETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    *project_id = attr.fsx_projid;
    close(fd);
    return 0;
}

int clear_project_id(const char *path) {
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        return errno;
    }

    struct fsxattr attr;
    memset(&attr, 0, sizeof(attr));

    if (ioctl(fd, FS_IOC_FSGETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    attr.fsx_projid = 0;

    if (ioctl(fd, FS_IOC_FSSETXATTR, &attr) < 0) {
        close(fd);
        return errno;
    }

    close(fd);
    return 0;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type XFSManager struct{}

func (m *XFSManager) SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error {
	return xfsSetQuota(path, id, int(qtype), bhard, bsoft, ihard, isoft)
}

func (m *XFSManager) GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error) {
	info, err := xfsGetQuota(path, id, int(qtype))
	if err != nil {
		return nil, err
	}
	return &QuotaInfo{
		ID:             info.ID,
		Type:           QuotaType(info.Type),
		BlockHardLimit: info.BlockHardLimit,
		BlockSoftLimit: info.BlockSoftLimit,
		CurrentBlocks:  info.CurrentBlocks,
		InodeHardLimit: info.InodeHardLimit,
		InodeSoftLimit: info.InodeSoftLimit,
		CurrentInodes:  info.CurrentInodes,
		BlockTime:      info.BlockTime,
		InodeTime:      info.InodeTime,
	}, nil
}

func (m *XFSManager) ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error) {
	infos, err := xfsListQuotas(path, int(qtype), maxID)
	if err != nil {
		return nil, err
	}
	result := make([]QuotaInfo, len(infos))
	for i, info := range infos {
		result[i] = QuotaInfo{
			ID:             info.ID,
			Type:           QuotaType(info.Type),
			BlockHardLimit: info.BlockHardLimit,
			BlockSoftLimit: info.BlockSoftLimit,
			CurrentBlocks:  info.CurrentBlocks,
			InodeHardLimit: info.InodeHardLimit,
			InodeSoftLimit: info.InodeSoftLimit,
			CurrentInodes:  info.CurrentInodes,
			BlockTime:      info.BlockTime,
			InodeTime:      info.InodeTime,
		}
	}
	return result, nil
}

func (m *XFSManager) RemoveQuota(path string, id uint32, qtype QuotaType) error {
	return xfsRemoveQuota(path, id, int(qtype))
}

func (m *XFSManager) TestQuota(path string, id uint32, qtype QuotaType) error {
	return xfsTestQuota(path, id, int(qtype))
}

type xfsQuotaInfo struct {
	ID             uint32
	Type           int32
	BlockHardLimit uint64
	BlockSoftLimit uint64
	CurrentBlocks  uint64
	InodeHardLimit uint64
	InodeSoftLimit uint64
	CurrentInodes  uint64
	BlockTime      uint64
	InodeTime      uint64
}

func xfsSetQuota(path string, id uint32, qtype int, bhard, bsoft, ihard, isoft uint64) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.xfs_set_quota(cPath, C.uint32_t(id), C.int(qtype),
		C.ulong(bhard), C.ulong(bsoft), C.ulong(ihard), C.ulong(isoft))

	if ret != 0 {
		errMsg := C.GoString(C.xfs_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}

func xfsGetQuota(path string, id uint32, qtype int) (*xfsQuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var info C.XFSQuotaInfo
	ret := C.xfs_get_quota(cPath, C.uint32_t(id), C.int(qtype), &info)

	if ret != 0 {
		errMsg := C.GoString(C.xfs_error_string(C.int(ret)))
		return nil, &QuotaError{Code: int(ret), Message: errMsg}
	}

	return &xfsQuotaInfo{
		ID:             uint32(info.id),
		Type:           int32(info.qtype),
		BlockHardLimit: uint64(info.bhardlimit),
		BlockSoftLimit: uint64(info.bsoftlimit),
		CurrentBlocks:  uint64(info.curblocks),
		InodeHardLimit: uint64(info.ihardlimit),
		InodeSoftLimit: uint64(info.isoftlimit),
		CurrentInodes:  uint64(info.curinodes),
		BlockTime:      uint64(info.btime),
		InodeTime:      uint64(info.itime),
	}, nil
}

func xfsListQuotas(path string, qtype int, maxID uint32) ([]xfsQuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var list C.XFSQuotaList
	ret := C.xfs_list_quotas(cPath, C.int(qtype), &list, C.int(maxID))

	if ret != 0 {
		errMsg := C.GoString(C.xfs_error_string(C.int(ret)))
		return nil, &QuotaError{Code: int(ret), Message: errMsg}
	}

	defer C.xfs_free_quota_list(&list)

	infos := make([]xfsQuotaInfo, int(list.count))
	for i := 0; i < int(list.count); i++ {
		item := (*[1 << 28]C.XFSQuotaInfo)(unsafe.Pointer(list.items))[i]
		infos[i] = xfsQuotaInfo{
			ID:             uint32(item.id),
			Type:           int32(item.qtype),
			BlockHardLimit: uint64(item.bhardlimit),
			BlockSoftLimit: uint64(item.bsoftlimit),
			CurrentBlocks:  uint64(item.curblocks),
			InodeHardLimit: uint64(item.ihardlimit),
			InodeSoftLimit: uint64(item.isoftlimit),
			CurrentInodes:  uint64(item.curinodes),
			BlockTime:      uint64(item.btime),
			InodeTime:      uint64(item.itime),
		}
	}

	return infos, nil
}

func xfsRemoveQuota(path string, id uint32, qtype int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.xfs_remove_quota(cPath, C.uint32_t(id), C.int(qtype))

	if ret != 0 {
		errMsg := C.GoString(C.xfs_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}

func xfsTestQuota(path string, id uint32, qtype int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.xfs_test_quota(cPath, C.uint32_t(id), C.int(qtype))

	if ret != 0 {
		errMsg := C.GoString(C.xfs_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}

// setProjectIDXFS 在XFS文件系统上设置project ID（使用ioctl系统调用）
func setProjectIDXFS(path string, projectID int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.set_project_id_xfs(cPath, C.uint32_t(projectID))
	if ret != 0 {
		errMsg := C.GoString(C.project_error_string(C.int(ret)))
		return fmt.Errorf("XFS设置失败: %s", errMsg)
	}
	return nil
}

// setProjectIDExt4 在ext4文件系统上设置project ID（使用ioctl系统调用）
func setProjectIDExt4(path string, projectID int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.set_project_id_ext4(cPath, C.uint32_t(projectID))
	if ret != 0 {
		errMsg := C.GoString(C.project_error_string(C.int(ret)))
		return fmt.Errorf("ext4设置失败: %s", errMsg)
	}
	return nil
}

// getProjectID 获取文件或目录的project ID（使用ioctl系统调用）
func getProjectID(path string) (int, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var projectID C.uint32_t
	ret := C.get_project_id(cPath, &projectID)
	if ret != 0 {
		errMsg := C.GoString(C.project_error_string(C.int(ret)))
		return 0, fmt.Errorf("获取project ID失败: %s", errMsg)
	}
	return int(projectID), nil
}

// clearProjectID 清除文件或目录的project ID（使用ioctl系统调用）
func clearProjectID(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.clear_project_id(cPath)
	if ret != 0 {
		errMsg := C.GoString(C.project_error_string(C.int(ret)))
		return fmt.Errorf("清除project ID失败: %s", errMsg)
	}
	return nil
}
