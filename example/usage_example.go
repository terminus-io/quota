package main

import (
	"fmt"
	"log"

	"github.com/terminus-io/quota"
)

func UsageExample() {
	path := "/"

	fmt.Println("=== Quota Library Usage Example ===")
	fmt.Println()

	fmt.Println("1. Detect filesystem type")
	fstype, err := quota.DetectFileSystem(path)
	if err != nil {
		log.Fatalf("Failed to detect filesystem: %v", err)
	}
	fmt.Printf("   Filesystem: %s\n", fstype)
	fmt.Println()

	fmt.Println("2. Set user quota for UID 1000")
	bhard := uint64(1048576)
	bsoft := uint64(921600)
	ihard := uint64(100000)
	isoft := uint64(90000)
	err = quota.SetQuota(path, 1000, quota.UserQuota, bhard, bsoft, ihard, isoft)
	if err != nil {
		log.Fatalf("Failed to set quota: %v", err)
	}
	fmt.Printf("   Block limits: hard=%d, soft=%d (1K blocks)\n", bhard, bsoft)
	fmt.Printf("   Inode limits: hard=%d, soft=%d\n", ihard, isoft)
	fmt.Println()

	fmt.Println("3. Get quota for UID 1000")
	info, err := quota.GetQuota(path, 1000, quota.UserQuota)
	if err != nil {
		log.Fatalf("Failed to get quota: %v", err)
	}
	fmt.Printf("   ID: %d\n", info.ID)
	fmt.Printf("   Block Hard Limit: %d (%.2f GB)\n", info.BlockHardLimit, float64(info.BlockHardLimit)/1024/1024)
	fmt.Printf("   Block Soft Limit: %d (%.2f GB)\n", info.BlockSoftLimit, float64(info.BlockSoftLimit)/1024/1024)
	fmt.Printf("   Current Blocks: %d (%.2f GB)\n", info.CurrentBlocks, float64(info.CurrentBlocks)/1024/1024)
	fmt.Printf("   Inode Hard Limit: %d\n", info.InodeHardLimit)
	fmt.Printf("   Inode Soft Limit: %d\n", info.InodeSoftLimit)
	fmt.Printf("   Current Inodes: %d\n", info.CurrentInodes)
	fmt.Println()

	fmt.Println("4. List all user quotas")
	infos, err := quota.ListQuotas(path, quota.UserQuota, 65536)
	if err != nil {
		log.Fatalf("Failed to list quotas: %v", err)
	}
	fmt.Printf("   Found %d quota(s)\n", len(infos))
	for _, q := range infos {
		fmt.Printf("   UID %d: blocks=%d/%d, inodes=%d/%d\n",
			q.ID, q.CurrentBlocks, q.BlockHardLimit, q.CurrentInodes, q.InodeHardLimit)
	}
	fmt.Println()

	fmt.Println("5. Test if quota exists for UID 1000")
	err = quota.TestQuota(path, 1000, quota.UserQuota)
	if err != nil {
		fmt.Printf("   Quota does not exist: %v\n", err)
	} else {
		fmt.Println("   Quota exists for UID 1000")
	}
	fmt.Println()

	fmt.Println("6. Remove quota for UID 1000")
	err = quota.RemoveQuota(path, 1000, quota.UserQuota)
	if err != nil {
		log.Fatalf("Failed to remove quota: %v", err)
	}
	fmt.Println("   Quota removed successfully")
	fmt.Println()

	fmt.Println("=== Example completed ===")
}
