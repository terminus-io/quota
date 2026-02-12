# Quota Library 使用指南

## 简介

这是一个用于管理 Linux 文件系统配额的 Go 库，支持 XFS 和 EXT4 文件系统。

## 版本选择

### 推荐版本：统一版本

统一版本同时支持 XFS 和 EXT4 文件系统，在运行时自动检测文件系统类型并选择相应的实现。

**优势:**
- 单一二进制文件，无需根据文件系统类型选择版本
- 运行时自动检测文件系统类型
- 适合容器化部署，每个节点可能使用不同文件系统

**编译:**
```bash
./build-unified.sh
```

**使用:**
```go
import "github.com/terminus-io/quota"

// 自动检测文件系统类型并使用相应的实现
err := quota.SetQuota("/data", 1000, quota.UserQuota, 1048576, 921600, 100000, 90000)
```

详细文档请查看 [UNIFIED.md](file:///home/pamforever/code/quota/UNIFIED.md)

### 专用版本

如果确定只使用一种文件系统，可以使用专用版本以减小二进制文件大小。

**XFS 版本:**
```bash
go build -tags xfs -o quota-tool ./cmd/quota-tool
```

**EXT4 版本:**
```bash
# 先编译 C 源文件
gcc -c -Wall -Wextra -I. pkg/ext4/quota_ext4.c -o pkg/ext4/quota_ext4.o

# 然后编译 Go 程序
go build -tags ext4 -o quota-tool ./cmd/quota-tool
```

## 安装

```bash
go get github.com/terminus-io/quota
```

## API 参考

### 配额类型

```go
const (
    UserQuota QuotaType = iota
    GroupQuota
    ProjQuota
)
```

### 主要函数

#### 1. 检测文件系统类型

```go
fstype, err := quota.DetectFileSystem("/path/to/mount")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Filesystem: %s\n", fstype)
```

#### 2. 设置配额

```go
err := quota.SetQuota(
    "/path/to/mount",           // 挂载点路径
    1000,                      // ID (UID/GID/Project ID)
    quota.ProjQuota,            // 配额类型
    1048576,                   // Block 硬限制 (1K blocks)
    921600,                    // Block 软限制 (1K blocks)
    100000,                    // Inode 硬限制
    90000,                     // Inode 软限制
)
```

#### 3. 获取配额信息

```go
info, err := quota.GetQuota("/path/to/mount", 1000, quota.ProjQuota)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("ID: %d\n", info.ID)
fmt.Printf("Block Hard Limit: %d\n", info.BlockHardLimit)
fmt.Printf("Block Soft Limit: %d\n", info.BlockSoftLimit)
fmt.Printf("Current Blocks: %d\n", info.CurrentBlocks)
fmt.Printf("Inode Hard Limit: %d\n", info.InodeHardLimit)
fmt.Printf("Inode Soft Limit: %d\n", info.InodeSoftLimit)
fmt.Printf("Current Inodes: %d\n", info.CurrentInodes)
```

#### 4. 列出所有配额

```go
infos, err := quota.ListQuotas("/path/to/mount", quota.ProjQuota, 65536)
if err != nil {
    log.Fatal(err)
}

for _, q := range infos {
    fmt.Printf("ID %d: blocks=%d/%d, inodes=%d/%d\n",
        q.ID, q.CurrentBlocks, q.BlockHardLimit, q.CurrentInodes, q.InodeHardLimit)
}
```

#### 5. 测试配额是否存在

```go
err := quota.TestQuota("/path/to/mount", 1000, quota.ProjQuota)
if err != nil {
    fmt.Println("Quota does not exist")
} else {
    fmt.Println("Quota exists")
}
```

#### 6. 删除配额

```go
err := quota.RemoveQuota("/path/to/mount", 1000, quota.ProjQuota)
if err != nil {
    log.Fatal(err)
}
```

## 完整示例

```go
package main

import (
    "fmt"
    "log"

    "github.com/terminus-io/quota"
)

func main() {
    path := "/data"

    // 1. 检测文件系统
    fstype, err := quota.DetectFileSystem(path)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Filesystem: %s\n", fstype)

    // 2. 设置配额
    err = quota.SetQuota(path, 1000, quota.UserQuota,
        1048576,  // 1GB hard limit (1K blocks)
        921600,   // 900MB soft limit (1K blocks)
        100000,    // 100K inodes hard limit
        90000,     // 90K inodes soft limit
    )
    if err != nil {
        log.Fatal(err)
    }

    // 3. 获取配额信息
    info, err := quota.GetQuota(path, 1000, quota.UserQuota)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Block usage: %.2f GB / %.2f GB\n",
        float64(info.CurrentBlocks)/1024/1024,
        float64(info.BlockHardLimit)/1024/1024)

    // 4. 列出所有配额
    quotas, err := quota.ListQuotas(path, quota.UserQuota, 65536)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d quota(s)\n", len(quotas))

    // 5. 删除配额
    err = quota.RemoveQuota(path, 1000, quota.UserQuota)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 注意事项

1. **权限要求**: 配额操作需要 root 权限
2. **文件系统支持**: 确保文件系统已启用配额功能
3. **单位说明**:
   - Block 限制单位为 1K blocks
   - 1GB = 1048576 1K blocks
4. **版本选择**: 推荐使用统一版本，自动适配不同文件系统

## 错误处理

所有函数都可能返回错误，建议始终检查错误：

```go
info, err := quota.GetQuota(path, id, quota.ProjQuota)
if err != nil {
    if err.Error() == "operation not permitted" {
        log.Fatal("需要 root 权限")
    } else if err.Error() == "no such device" {
        log.Fatal("文件系统不支持配额")
    } else {
        log.Fatal(err)
    }
}
```

## 更多文档

- [统一版本详细指南](UNIFIED.md)
- [示例代码](example/usage_example.go)
- [README](README.md)
