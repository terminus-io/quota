package quota

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

type QuotaType int

const (
	UserQuota  QuotaType = 0
	GroupQuota QuotaType = 1
	ProjQuota  QuotaType = 2
)

type FileSystemType string

const (
	FileSystemXFS  FileSystemType = "xfs"
	FileSystemEXT4 FileSystemType = "ext4"
)

type QuotaInfo struct {
	ID             uint32
	Type           QuotaType
	BlockHardLimit uint64
	BlockSoftLimit uint64
	CurrentBlocks  uint64
	InodeHardLimit uint64
	InodeSoftLimit uint64
	CurrentInodes  uint64
	BlockTime      uint64
	InodeTime      uint64
}

type QuotaError struct {
	Code    int
	Message string
}

func (e *QuotaError) Error() string {
	return fmt.Sprintf("quota error (code %d): %s", e.Code, e.Message)
}

type QuotaManager interface {
	SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error
	GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error)
	ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error)
	RemoveQuota(path string, id uint32, qtype QuotaType) error
	TestQuota(path string, id uint32, qtype QuotaType) error
}

func DetectFileSystem(path string) (FileSystemType, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return "", err
	}

	fstype := stat.Type
	switch fstype {
	case 0x58465342:
		return FileSystemXFS, nil
	case 0xEF53:
		return FileSystemEXT4, nil
	default:
		return "", fmt.Errorf("unsupported filesystem type: 0x%x", fstype)
	}
}

func DetectFileSystemByCommand(path string) (FileSystemType, error) {
	cmd := exec.Command("df", "-T", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("invalid df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid df output format")
	}

	fstype := strings.ToLower(fields[1])
	if strings.Contains(fstype, "xfs") {
		return FileSystemXFS, nil
	} else if strings.Contains(fstype, "ext4") {
		return FileSystemEXT4, nil
	}

	return "", fmt.Errorf("unsupported filesystem: %s", fstype)
}

func SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error {
	mgr, err := NewQuotaManager(path)
	if err != nil {
		return err
	}
	return mgr.SetQuota(path, id, qtype, bhard, bsoft, ihard, isoft)
}

func GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error) {
	mgr, err := NewQuotaManager(path)
	if err != nil {
		return nil, err
	}
	return mgr.GetQuota(path, id, qtype)
}

func ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error) {
	mgr, err := NewQuotaManager(path)
	if err != nil {
		return nil, err
	}
	return mgr.ListQuotas(path, qtype, maxID)
}

func RemoveQuota(path string, id uint32, qtype QuotaType) error {
	mgr, err := NewQuotaManager(path)
	if err != nil {
		return err
	}
	return mgr.RemoveQuota(path, id, qtype)
}

func TestQuota(path string, id uint32, qtype QuotaType) error {
	mgr, err := NewQuotaManager(path)
	if err != nil {
		return err
	}
	return mgr.TestQuota(path, id, qtype)
}
