# Quota Library for Linux

A Go library for managing disk quotas on Linux filesystems (XFS and EXT4).

## Features

- **Unified Interface**: Single API for both XFS and EXT4 filesystems
- **Automatic Detection**: Automatically detects filesystem type
- **Project Quota Support**: Full support for project quotas (XFS PRJQUOTA)
- **User/Group Quota**: Support for user and group quotas
- **Type Safety**: Full Go type safety with CGO bindings

## Installation

### Prerequisites

#### XFS Support
```bash
# CentOS 7
yum install -y gcc make glibc-devel xfsprogs-devel quota-devel

# CentOS 8/9
dnf install -y gcc make glibc-devel xfsprogs-devel quota-devel
```

#### EXT4 Support
```bash
# CentOS 7
yum install -y gcc make glibc-devel quota-devel

# CentOS 8/9
dnf install -y gcc make glibc-devel quota-devel
```

### Build

```bash
go build -o quota-tool ./cmd/quota-tool
```

## Usage

### As a Library

```go
package main

import (
	"fmt"
	"log"
	"github.com/terminus-io/quota"
)

func main() {
	path := "/mnt/data"
	id := uint32(1000)

	// Detect filesystem and create manager
	mgr, err := quota.NewQuotaManager(path)
	if err != nil {
		log.Fatal(err)
	}

	// Set quota
	err = mgr.SetQuota(path, id, quota.ProjQuota, 
		10485760, 10485760, 100000, 90000)
	if err != nil {
		log.Fatal(err)
	}

	// Get quota
	info, err := mgr.GetQuota(path, id, quota.ProjQuota)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Block Limit: %d GB\n", info.BlockHardLimit/1024/1024)

	// List quotas
	infos, err := mgr.ListQuotas(path, quota.ProjQuota, 65536)
	if err != nil {
		log.Fatal(err)
	}

	for _, q := range infos {
		fmt.Printf("ID %d: %d GB\n", q.ID, q.BlockHardLimit/1024/1024)
	}

	// Remove quota
	err = mgr.RemoveQuota(path, id, quota.ProjQuota)
	if err != nil {
		log.Fatal(err)
	}
}
```

### Command Line Tool

```bash
# Detect filesystem type
./quota-tool detect /mnt/data

# Set quota
./quota-tool set /mnt/data 1000 10485760 10485760 100000 90000

# Get quota
./quota-tool get /mnt/data 1000

# List quotas
./quota-tool list /mnt/data project 65536

# Remove quota
./quota-tool remove /mnt/data 1000
```

## API Reference

### Types

#### QuotaType
```go
const (
	UserQuota  QuotaType = 0
	GroupQuota QuotaType = 1
	ProjQuota  QuotaType = 2
)
```

#### FileSystemType
```go
const (
	FileSystemXFS  FileSystemType = "xfs"
	FileSystemEXT4 FileSystemType = "ext4"
)
```

#### QuotaInfo
```go
type QuotaInfo struct {
	ID              uint32
	Type            QuotaType
	BlockHardLimit  uint64
	BlockSoftLimit  uint64
	CurrentBlocks   uint64
	InodeHardLimit  uint64
	InodeSoftLimit  uint64
	CurrentInodes   uint64
	BlockTime       uint64
	InodeTime       uint64
}
```

#### QuotaManager
```go
type QuotaManager interface {
	SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error
	GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error)
	ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error)
	RemoveQuota(path string, id uint32, qtype QuotaType) error
	TestQuota(path string, id uint32, qtype QuotaType) error
}
```

### Functions

#### DetectFileSystem
Detects the filesystem type for a given path.

```go
func DetectFileSystem(path string) (FileSystemType, error)
```

#### NewQuotaManager
Creates a quota manager for the filesystem type detected at the given path.

```go
func NewQuotaManager(path string) (QuotaManager, error)
```

#### NewQuotaManagerForType
Creates a quota manager for a specific filesystem type.

```go
func NewQuotaManagerForType(fstype FileSystemType) (QuotaManager, error)
```

#### SetQuota
Sets quota limits for a given ID.

```go
func SetQuota(path string, id uint32, qtype QuotaType, bhard, bsoft, ihard, isoft uint64) error
```

Parameters:
- `path`: Mount point path
- `id`: Quota ID (UID/GID/Project ID)
- `qtype`: Quota type (UserQuota/GroupQuota/ProjQuota)
- `bhard`: Block hard limit (1K blocks)
- `bsoft`: Block soft limit (1K blocks)
- `ihard`: Inode hard limit
- `isoft`: Inode soft limit

#### GetQuota
Gets quota information for a given ID.

```go
func GetQuota(path string, id uint32, qtype QuotaType) (*QuotaInfo, error)
```

#### ListQuotas
Lists all quotas of a given type.

```go
func ListQuotas(path string, qtype QuotaType, maxID uint32) ([]QuotaInfo, error)
```

Parameters:
- `path`: Mount point path
- `qtype`: Quota type (UserQuota/GroupQuota/ProjQuota)
- `maxID`: Maximum ID to search (default: 65536)

#### RemoveQuota
Removes quota limits for a given ID.

```go
func RemoveQuota(path string, id uint32, qtype QuotaType) error
```

#### TestQuota
Tests if a quota exists for a given ID.

```go
func TestQuota(path string, id uint32, qtype QuotaType) error
```

## Filesystem Differences

### XFS
- Uses `Q_XSETQLIM` and `Q_XGETQUOTA` quotactl commands
- Supports project quotas (PRJQUOTA)
- Block units: 512-byte blocks (converted to 1K blocks in API)
- Requires `xfsprogs-devel` package

### EXT4
- Uses `Q_SETQUOTA` and `Q_GETQUOTA` quotactl commands
- Supports user and group quotas
- Block units: 1K blocks
- Requires `quota-devel` package

## Examples

### Project Quota on XFS

```go
mgr, _ := quota.NewQuotaManager("/mnt/data")
mgr.SetQuota("/mnt/data", 1000, quota.ProjQuota, 
	10*1024*1024, 10*1024*1024, 100000, 90000)
```

### User Quota on EXT4

```go
mgr, _ := quota.NewQuotaManager("/home")
mgr.SetQuota("/home", 1000, quota.UserQuota, 
	10*1024*1024, 10*1024*1024, 100000, 90000)
```

### Automatic Filesystem Detection

```go
fstype, err := quota.DetectFileSystem("/mnt/data")
if err != nil {
	log.Fatal(err)
}

switch fstype {
case quota.FileSystemXFS:
	fmt.Println("XFS filesystem detected")
case quota.FileSystemEXT4:
	fmt.Println("EXT4 filesystem detected")
}
```

## License

MIT License
