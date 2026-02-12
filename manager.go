package quota

import (
	"fmt"
)

func NewQuotaManager(path string) (QuotaManager, error) {
	fstype, err := DetectFileSystem(path)
	if err != nil {
		return nil, err
	}

	return NewQuotaManagerForType(fstype)
}

func NewQuotaManagerForType(fstype FileSystemType) (QuotaManager, error) {
	switch fstype {
	case FileSystemXFS:
		return &XFSManager{}, nil
	case FileSystemEXT4:
		return &EXT4Manager{}, nil
	default:
		return nil, fmt.Errorf("unsupported filesystem: %s", fstype)
	}
}
