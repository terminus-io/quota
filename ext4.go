package quota

/*
#cgo CFLAGS: -Wall -Wextra -I${SRCDIR}/pkg/ext4
#cgo LDFLAGS: ${SRCDIR}/pkg/ext4/quota_ext4.o ${SRCDIR}/pkg/ext4/quota_ext4_fast.o ${SRCDIR}/pkg/ext4/quota_ext4_direct.o
#include <stdlib.h>
#include "quota_ext4.h"
*/
import "C"

import (
	"unsafe"
)

type EXT4Manager struct{}

func (m *EXT4Manager) SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error {
	return ext4SetQuota(path, id, int(qtype), bhard, bsoft, ihard, isoft)
}

func (m *EXT4Manager) GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error) {
	info, err := ext4GetQuota(path, id, int(qtype))
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

func (m *EXT4Manager) ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error) {
	infos, err := ext4ListQuotasDirect(path, int(qtype), maxID)
	if err == nil {
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

	infos, err = ext4ListQuotasFast(path, int(qtype), maxID)
	if err == nil {
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

	infos, err = ext4ListQuotas(path, int(qtype), maxID)
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

func (m *EXT4Manager) RemoveQuota(path string, id uint32, qtype QuotaType) error {
	return ext4RemoveQuota(path, id, int(qtype))
}

func (m *EXT4Manager) TestQuota(path string, id uint32, qtype QuotaType) error {
	return ext4TestQuota(path, id, int(qtype))
}

type ext4QuotaInfo struct {
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

func ext4SetQuota(path string, id uint32, qtype int, bhard, bsoft, ihard, isoft uint64) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.ext4_set_quota(cPath, C.uint32_t(id), C.int(qtype),
		C.ulong(bhard), C.ulong(bsoft), C.ulong(ihard), C.ulong(isoft))

	if ret != 0 {
		errMsg := C.GoString(C.ext4_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}

func ext4GetQuota(path string, id uint32, qtype int) (*ext4QuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var info C.EXT4QuotaInfo
	ret := C.ext4_get_quota(cPath, C.uint32_t(id), C.int(qtype), &info)

	if ret != 0 {
		errMsg := C.GoString(C.ext4_error_string(C.int(ret)))
		return nil, &QuotaError{Code: int(ret), Message: errMsg}
	}

	return &ext4QuotaInfo{
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

func ext4ListQuotas(path string, qtype int, maxID uint32) ([]ext4QuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var list C.EXT4QuotaList
	ret := C.ext4_list_quotas(cPath, C.int(qtype), &list, C.int(maxID))

	if ret != 0 {
		errMsg := C.GoString(C.ext4_error_string(C.int(ret)))
		return nil, &QuotaError{Code: int(ret), Message: errMsg}
	}

	defer C.ext4_free_quota_list(&list)

	infos := make([]ext4QuotaInfo, int(list.count))
	for i := 0; i < int(list.count); i++ {
		item := (*[1 << 28]C.EXT4QuotaInfo)(unsafe.Pointer(list.items))[i]
		infos[i] = ext4QuotaInfo{
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

func ext4ListQuotasFast(path string, qtype int, maxID uint32) ([]ext4QuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var list C.EXT4QuotaList
	ret := C.ext4_list_quotas_fast(cPath, C.int(qtype), &list, C.int(maxID))

	if ret == 0 {
		defer C.ext4_free_quota_list(&list)

		infos := make([]ext4QuotaInfo, int(list.count))
		for i := 0; i < int(list.count); i++ {
			item := (*[1 << 28]C.EXT4QuotaInfo)(unsafe.Pointer(list.items))[i]
			infos[i] = ext4QuotaInfo{
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

	return nil, &QuotaError{Code: int(ret), Message: "Fast method not available"}
}

func ext4ListQuotasDirect(path string, qtype int, maxID uint32) ([]ext4QuotaInfo, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var list C.EXT4QuotaList
	ret := C.ext4_list_quotas_direct(cPath, C.int(qtype), &list, C.int(maxID))

	if ret == 0 {
		defer C.ext4_free_quota_list(&list)

		infos := make([]ext4QuotaInfo, int(list.count))
		for i := 0; i < int(list.count); i++ {
			item := (*[1 << 28]C.EXT4QuotaInfo)(unsafe.Pointer(list.items))[i]
			infos[i] = ext4QuotaInfo{
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

	return nil, &QuotaError{Code: int(ret), Message: "Direct method not available"}
}

func ext4RemoveQuota(path string, id uint32, qtype int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.ext4_remove_quota(cPath, C.uint32_t(id), C.int(qtype))

	if ret != 0 {
		errMsg := C.GoString(C.ext4_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}

func ext4TestQuota(path string, id uint32, qtype int) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	ret := C.ext4_test_quota(cPath, C.uint32_t(id), C.int(qtype))

	if ret != 0 {
		errMsg := C.GoString(C.ext4_error_string(C.int(ret)))
		return &QuotaError{Code: int(ret), Message: errMsg}
	}

	return nil
}
