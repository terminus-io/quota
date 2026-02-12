package quota

/*
#cgo CFLAGS: -Wall -Wextra
#include <stdlib.h>
#include "pkg/xfs/quota_xfs.h"
*/
import "C"

import (
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
