package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/terminus-io/quota"
)

func printUsage() {
	fmt.Println("Quota Management Tool (XFS & EXT4)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  quota-tool <command> <path> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  set          Set quota limits")
	fmt.Println("  set-project  Set project ID for a path")
	fmt.Println("  get          Get quota information for a specific ID")
	fmt.Println("  list         List all quotas of a given type")
	fmt.Println("  test-id      Test if a specific ID has quota")
	fmt.Println("  remove       Remove quota limits")
	fmt.Println("  test         Run comprehensive tests")
	fmt.Println("  detect       Detect filesystem type")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  Detect filesystem:")
	fmt.Println("    quota-tool detect /mnt/data")
	fmt.Println()
	fmt.Println("  Set project ID:")
	fmt.Println("    quota-tool set-project /mnt/data 100")
	fmt.Println()
	fmt.Println("  Set quota:")
	fmt.Println("    quota-tool set /mnt/data 1000 1048576 921600 100000 90000")
	fmt.Println("    (path id bhard bsoft ihard isoft)")
	fmt.Println()
	fmt.Println("  Get quota:")
	fmt.Println("    quota-tool get /mnt/data 1000")
	fmt.Println()
	fmt.Println("  Test quota ID:")
	fmt.Println("    quota-tool test-id /mnt/data project 20121")
	fmt.Println()
	fmt.Println("  List quotas:")
	fmt.Println("    quota-tool list /mnt/data user [max_id]")
	fmt.Println("    quota-tool list /mnt/data group [max_id]")
	fmt.Println("    quota-tool list /mnt/data project [max_id]")
	fmt.Println()
	fmt.Println("  Remove quota:")
	fmt.Println("    quota-tool remove /mnt/data 1000")
	fmt.Println()
	fmt.Println("  Run tests:")
	fmt.Println("    quota-tool test /mnt/data 1000")
}

func setQuota(path string, args []string) {
	if len(args) < 5 {
		log.Fatal("set command requires: id bhard bsoft ihard isoft")
	}

	id, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		log.Fatalf("Invalid id value: %v", err)
	}

	bhard, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid bhard value: %v", err)
	}

	bsoft, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		log.Fatalf("Invalid bsoft value: %v", err)
	}

	ihard, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		log.Fatalf("Invalid ihard value: %v", err)
	}

	isoft, err := strconv.ParseUint(args[4], 10, 64)
	if err != nil {
		log.Fatalf("Invalid isoft value: %v", err)
	}

	fmt.Printf("Setting quota for path=%s, id=%d\n", path, id)
	fmt.Printf("  Block limits: soft=%d, hard=%d (1K blocks)\n", bsoft, bhard)
	fmt.Printf("  Inode limits: soft=%d, hard=%d\n", isoft, ihard)

	err = quota.SetQuota(path, uint32(id), quota.ProjQuota, bhard, bsoft, ihard, isoft)
	if err != nil {
		log.Fatalf("Failed to set quota: %v", err)
	}

	fmt.Println("✓ Quota set successfully")
}

func getQuota(path string, uidStr string) {
	uid, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid uid value: %v", err)
	}

	fmt.Printf("Getting quota for path=%s, uid=%d\n", path, uid)

	info, err := quota.GetQuota(path, uint32(uid), quota.ProjQuota)
	if err != nil {
		log.Fatalf("Failed to get quota: %v", err)
	}

	fmt.Println()
	fmt.Println("Quota Information:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("ID:              %d\n", info.ID)
	fmt.Printf("Type:            %d\n", info.Type)
	fmt.Printf("Block Hard Limit: %d 1K blocks (%.2f GB)\n",
		info.BlockHardLimit, float64(info.BlockHardLimit)/1024/1024)
	fmt.Printf("Block Soft Limit: %d 1K blocks (%.2f GB)\n",
		info.BlockSoftLimit, float64(info.BlockSoftLimit)/1024/1024)
	fmt.Printf("Current Blocks:   %d 1K blocks (%.2f GB)\n",
		info.CurrentBlocks, float64(info.CurrentBlocks)/1024/1024)
	fmt.Println()
	fmt.Printf("Inode Hard Limit: %d\n", info.InodeHardLimit)
	fmt.Printf("Inode Soft Limit: %d\n", info.InodeSoftLimit)
	fmt.Printf("Current Inodes:   %d\n", info.CurrentInodes)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func removeQuota(path string, uidStr string) {
	uid, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid uid value: %v", err)
	}

	fmt.Printf("Removing quota for path=%s, uid=%d\n", path, uid)

	err = quota.RemoveQuota(path, uint32(uid), quota.ProjQuota)
	if err != nil {
		log.Fatalf("Failed to remove quota: %v", err)
	}

	fmt.Println("✓ Quota removed successfully")
}

func listQuotas(path string, args []string) {
	if len(args) < 1 {
		log.Fatal("list command requires: type [max_id]")
	}

	var qtype quota.QuotaType
	switch args[0] {
	case "user", "usr", "u":
		qtype = quota.UserQuota
	case "group", "grp", "g":
		qtype = quota.GroupQuota
	case "project", "proj", "p":
		qtype = quota.ProjQuota
	default:
		log.Fatalf("Invalid quota type: %s (must be user, group, or project)", args[0])
	}

	maxID := uint32(65536)
	if len(args) > 1 {
		parsed, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			log.Fatalf("Invalid max_id value: %v", err)
		}
		maxID = uint32(parsed)
	}

	fmt.Printf("Listing quotas for path=%s, type=%s, max_id=%d\n", path, args[0], maxID)

	infos, err := quota.ListQuotas(path, qtype, maxID)
	if err != nil {
		log.Fatalf("Failed to list quotas: %v", err)
	}

	if len(infos) == 0 {
		fmt.Println("No quotas found")
		return
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           Quota List                                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-8s ║ %-12s ║ %-12s ║ %-12s ║ %-12s ║\n", "ID", "Block Used", "Block Limit", "Inode Used", "Inode Limit")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	for _, info := range infos {
		blockLimit := info.BlockHardLimit
		if blockLimit == 0 {
			blockLimit = info.BlockSoftLimit
		}
		inodeLimit := info.InodeHardLimit
		if inodeLimit == 0 {
			inodeLimit = info.InodeSoftLimit
		}

		blockUsedStr := fmt.Sprintf("%.2f GB", float64(info.CurrentBlocks)/1024/1024)
		blockLimitStr := fmt.Sprintf("%.2f GB", float64(blockLimit)/1024/1024)
		if blockLimit == 0 {
			blockLimitStr = "unlimited"
		}

		fmt.Printf("║ %-8d ║ %-12s ║ %-12s ║ %-12d ║ %-12d ║\n",
			info.ID, blockUsedStr, blockLimitStr, info.CurrentInodes, inodeLimit)
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nTotal: %d quota(s) found\n", len(infos))
}

func runTests(path string, uidStr string) {
	uid, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid uid value: %v", err)
	}

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║        XFS Quota Library Test Suite                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	testsPassed := 0
	testsFailed := 0

	fmt.Println("Test 1: Set quota limits")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	err = quota.SetQuota(path, uint32(uid), quota.ProjQuota,
		1048576, 921600, 100000, 90000)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n\n", err)
		testsFailed++
	} else {
		fmt.Println("✓ PASSED")
		testsPassed++
	}
	fmt.Println()

	fmt.Println("Test 2: Get quota information")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	info, err := quota.GetQuota(path, uint32(uid), quota.ProjQuota)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n\n", err)
		testsFailed++
	} else {
		fmt.Println("✓ PASSED")
		fmt.Printf("  Retrieved info: blocks=%d/%d, inodes=%d/%d\n",
			info.CurrentBlocks, info.BlockHardLimit,
			info.CurrentInodes, info.InodeHardLimit)
		testsPassed++
	}
	fmt.Println()

	fmt.Println("Test 3: Verify quota values")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if info != nil {
		if info.BlockHardLimit == 1048576 &&
			info.BlockSoftLimit == 921600 &&
			info.InodeHardLimit == 100000 &&
			info.InodeSoftLimit == 90000 {
			fmt.Println("✓ PASSED - Values match expected limits")
			testsPassed++
		} else {
			fmt.Println("✗ FAILED - Values don't match")
			fmt.Printf("  Expected: bhard=1048576, bsoft=921600, ihard=100000, isoft=90000\n")
			fmt.Printf("  Got:      bhard=%d, bsoft=%d, ihard=%d, isoft=%d\n",
				info.BlockHardLimit, info.BlockSoftLimit,
				info.InodeHardLimit, info.InodeSoftLimit)
			testsFailed++
		}
	} else {
		fmt.Println("✗ FAILED - No quota info available")
		testsFailed++
	}
	fmt.Println()

	fmt.Println("Test 4: Update quota limits")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	err = quota.SetQuota(path, uint32(uid), quota.ProjQuota,
		2097152, 2048000, 200000, 180000)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n\n", err)
		testsFailed++
	} else {
		fmt.Println("✓ PASSED")
		testsPassed++
	}
	fmt.Println()

	fmt.Println("Test 5: Verify updated quota")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	info2, err := quota.GetQuota(path, uint32(uid), quota.ProjQuota)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n\n", err)
		testsFailed++
	} else {
		if info2.BlockHardLimit == 2097152 {
			fmt.Println("✓ PASSED - Quota updated successfully")
			testsPassed++
		} else {
			fmt.Println("✗ FAILED - Quota not updated correctly")
			testsFailed++
		}
	}
	fmt.Println()

	fmt.Println("Test 6: Remove quota")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	err = quota.RemoveQuota(path, uint32(uid), quota.ProjQuota)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n\n", err)
		testsFailed++
	} else {
		fmt.Println("✓ PASSED")
		testsPassed++
	}
	fmt.Println()

	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Printf("║  Test Results: %d passed, %d failed                        ║\n", testsPassed, testsFailed)
	fmt.Println("╚══════════════════════════════════════════════════════╝")
}

func testQuotaID(path string, typeStr string, idStr string) {
	var qtype quota.QuotaType
	switch typeStr {
	case "user", "usr", "u":
		qtype = quota.UserQuota
	case "group", "grp", "g":
		qtype = quota.GroupQuota
	case "project", "proj", "p":
		qtype = quota.ProjQuota
	default:
		log.Fatalf("Invalid quota type: %s (must be user, group, or project)", typeStr)
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid id value: %v", err)
	}

	fmt.Printf("Testing quota for path=%s, type=%s, id=%d\n", path, typeStr, id)

	err = quota.TestQuota(path, uint32(id), qtype)
	if err != nil {
		fmt.Printf("✗ FAILED: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("✓ PASSED - Quota exists for this ID")
	}
}

func detectFileSystem(path string) {
	fstype, err := quota.DetectFileSystem(path)
	if err != nil {
		log.Fatalf("Failed to detect filesystem: %v", err)
	}

	fmt.Printf("Filesystem: %s\n", fstype)
}

func setProjectID(path string, projectIDStr string) {
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid project ID value: %v", err)
	}

	fmt.Printf("Setting project ID %d for path=%s\n", projectID, path)

	err = quota.SetProjectID(path, int(projectID))
	if err != nil {
		log.Fatalf("Failed to set project ID: %v", err)
	}

	fmt.Println("✓ Project ID set successfully")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "set":
		if len(os.Args) < 7 {
			fmt.Println("Usage: quota-tool set <path> <id> <bhard> <bsoft> <ihard> <isoft>")
			os.Exit(1)
		}
		setQuota(os.Args[2], os.Args[3:])

	case "set-project":
		if len(os.Args) < 4 {
			fmt.Println("Usage: quota-tool set-project <path> <project_id>")
			os.Exit(1)
		}
		setProjectID(os.Args[2], os.Args[3])

	case "get":
		if len(os.Args) < 4 {
			fmt.Println("Usage: quota-tool get <path> <uid>")
			os.Exit(1)
		}
		getQuota(os.Args[2], os.Args[3])

	case "test-id":
		if len(os.Args) < 5 {
			fmt.Println("Usage: quota-tool test-id <path> <type> <id>")
			os.Exit(1)
		}
		testQuotaID(os.Args[2], os.Args[3], os.Args[4])

	case "list":
		if len(os.Args) < 4 {
			fmt.Println("Usage: quota-tool list <path> <type> [max_id]")
			os.Exit(1)
		}
		listQuotas(os.Args[2], os.Args[3:])

	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: quota-tool remove <path> <uid>")
			os.Exit(1)
		}
		removeQuota(os.Args[2], os.Args[3])

	case "test":
		if len(os.Args) < 4 {
			fmt.Println("Usage: quota-tool test <path> <uid>")
			os.Exit(1)
		}
		runTests(os.Args[2], os.Args[3])

	case "detect":
		if len(os.Args) < 3 {
			fmt.Println("Usage: quota-tool detect <path>")
			os.Exit(1)
		}
		detectFileSystem(os.Args[2])

	case "-h", "--help", "help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
