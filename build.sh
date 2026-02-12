#!/bin/bash

echo "Building unified binary for Linux (XFS + EXT4)..."

# 编译 EXT4 C 源文件
echo "Compiling EXT4 C sources..."
gcc -c -Wall -Wextra -I. pkg/ext4/quota_ext4.c -o pkg/ext4/quota_ext4.o
if [ $? -ne 0 ]; then
    echo "Failed to compile EXT4 C sources"
    exit 1
fi

# 构建统一的 Go 二进制文件
echo "Building unified Go binary..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build \
    -tags netgo \
    -ldflags "-linkmode external -extldflags '-static -Wl,--unresolved-symbols=ignore-in-shared-libs'" \
    -o quota-tool-unified \
    ./cmd/quota-tool

if [ $? -eq 0 ]; then
    echo "Build successful!"
    ls -lh quota-tool-unified
    file quota-tool-unified
else
    echo "Build failed!"
    exit 1
fi
