# 统一版本使用指南

## 简介

统一版本同时支持 XFS 和 EXT4 文件系统，在运行时自动检测文件系统类型并选择相应的实现，无需在编译时指定标签。

## 优势

- **单一二进制文件**: 一个二进制文件支持多种文件系统
- **自动检测**: 运行时自动检测文件系统类型
- **容器友好**: 适合在不同文件系统的容器环境中部署
- **简化部署**: 无需为不同文件系统构建不同版本

## 编译

### 统一版本（推荐）

```bash
chmod +x build-unified.sh
./build-unified.sh
```

这将生成 `quota-tool-unified` 二进制文件，同时支持 XFS 和 EXT4。

### 静态链接

```bash
# 编译 EXT4 C 源文件
gcc -c -Wall -Wextra -I. pkg/ext4/quota_ext4.c -o pkg/ext4/quota_ext4.o

# 构建统一版本
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -tags netgo \
    -ldflags "-linkmode external -extldflags '-static -Wl,--unresolved-symbols=ignore-in-shared-libs'" \
    -o your-app ./your-app
```

## 使用示例

### 在你的项目中使用

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/terminus-io/quota"
)

func main() {
    path := "/data"
    
    // 自动检测文件系统类型并选择合适的实现
    err := quota.SetQuota(path, 1000, quota.UserQuota,
        1048576,  // 1GB 硬限制
        921600,   // 900MB 软限制
        100000,    // 100K inodes 硬限制
        90000,     // 90K inodes 软限制
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取配额信息
    info, err := quota.GetQuota(path, 1000, quota.UserQuota)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Block usage: %.2f GB / %.2f GB\n",
        float64(info.CurrentBlocks)/1024/1024,
        float64(info.BlockHardLimit)/1024/1024)
}
```

### 手动检测文件系统

```go
// 自动检测文件系统类型
fstype, err := quota.DetectFileSystem("/data")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Filesystem: %s\n", fstype)
```

## 容器化部署

### Dockerfile 示例

```dockerfile
FROM golang:1.21 AS builder

WORKDIR /app
COPY . .

# 编译统一版本
RUN gcc -c -Wall -Wextra -I. pkg/ext4/quota_ext4.c -o pkg/ext4/quota_ext4.o
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -tags netgo \
    -ldflags "-linkmode external -extldflags '-static -Wl,--unresolved-symbols=ignore-in-shared-libs'" \
    -o quota-tool ./cmd/quota-tool

FROM alpine:latest
COPY --from=builder /app/quota-tool /usr/local/bin/

# 测试
RUN quota-tool detect /
```

### Kubernetes 部署

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: quota-manager
spec:
  containers:
  - name: quota-manager
    image: your-registry/quota-tool:latest
    command: ["/bin/sh"]
    args: ["-c", "quota-tool detect /data"]
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    hostPath:
      path: /var/lib/data
```

## 工作原理

统一版本通过以下方式支持多种文件系统：

1. **编译时**: 同时编译 XFS 和 EXT4 的 C 实现
2. **运行时**: 调用 `DetectFileSystem()` 检测文件系统类型
3. **动态选择**: 根据文件系统类型选择相应的管理器实现

```go
// quota.go 中的实现
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
```

## 注意事项

1. **权限要求**: 配额操作需要 root 权限
2. **文件系统支持**: 确保文件系统已启用配额功能
3. **二进制大小**: 统一版本比单一版本稍大，但功能更完整
4. **兼容性**: 统一版本可以替代之前的 XFS 或 EXT4 专用版本

## 迁移指南

### 从专用版本迁移

如果你之前使用的是专用版本：

**之前:**
```bash
# 需要根据文件系统选择版本
go build -tags xfs -o quota-tool ./cmd/quota-tool
# 或
go build -tags ext4 -o quota-tool ./cmd/quota-tool
```

**现在:**
```bash
# 统一版本，自动适配
./build-unified.sh
```

代码无需修改，API 完全兼容。
