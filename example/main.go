package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/terminus-io/quota"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  Set quota:   example set <path> <uid> <bhard> <bsoft> <ihard> <isoft>")
		fmt.Println("  Get quota:   example get <path> <uid>")
		fmt.Println("  Remove quota: example remove <path> <uid>")
		os.Exit(1)
	}

	command := os.Args[1]
	path := os.Args[2]

	switch command {
	case "set":
		if len(os.Args) < 8 {
			log.Fatal("set command requires: path uid bhard bsoft ihard isoft")
		}
		uid, _ := strconv.ParseUint(os.Args[3], 10, 32)
		bhard, _ := strconv.ParseUint(os.Args[4], 10, 64)
		bsoft, _ := strconv.ParseUint(os.Args[5], 10, 64)
		ihard, _ := strconv.ParseUint(os.Args[6], 10, 64)
		isoft, _ := strconv.ParseUint(os.Args[7], 10, 64)

		err := quota.SetQuota(path, uint32(uid), quota.UserQuota, bhard, bsoft, ihard, isoft)
		if err != nil {
			log.Fatalf("Failed to set quota: %v", err)
		}
		fmt.Println("Quota set successfully")

	case "get":
		if len(os.Args) < 4 {
			log.Fatal("get command requires: path uid")
		}
		uid, _ := strconv.ParseUint(os.Args[3], 10, 32)

		info, err := quota.GetQuota(path, uint32(uid), quota.UserQuota)
		if err != nil {
			log.Fatalf("Failed to get quota: %v", err)
		}
		fmt.Printf("Quota Info for UID %d:\n", uid)
		fmt.Printf("  Block Hard Limit: %d\n", info.BlockHardLimit)
		fmt.Printf("  Block Soft Limit: %d\n", info.BlockSoftLimit)
		fmt.Printf("  Current Blocks:   %d\n", info.CurrentBlocks)
		fmt.Printf("  Inode Hard Limit: %d\n", info.InodeHardLimit)
		fmt.Printf("  Inode Soft Limit: %d\n", info.InodeSoftLimit)
		fmt.Printf("  Current Inodes:   %d\n", info.CurrentInodes)

	case "remove":
		if len(os.Args) < 4 {
			log.Fatal("remove command requires: path uid")
		}
		uid, _ := strconv.ParseUint(os.Args[3], 10, 32)

		err := quota.RemoveQuota(path, uint32(uid), quota.UserQuota)
		if err != nil {
			log.Fatalf("Failed to remove quota: %v", err)
		}
		fmt.Println("Quota removed successfully")

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
