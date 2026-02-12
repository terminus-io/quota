package quota

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GetFilesystemType 获取指定路径的文件系统类型
func GetFilesystemType(path string) (string, error) {
	cmd := exec.Command("df", "-T", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("获取文件系统类型失败: %w", err)
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("无法解析文件系统类型")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return "", fmt.Errorf("无法解析文件系统类型")
	}

	return fields[1], nil
}

// SetProjectIDXFS 在XFS文件系统上设置project ID
func SetProjectIDXFS(path string, projectID int) error {
	cmd := exec.Command("xfs_quota", "-x", "-c", fmt.Sprintf("project -s -p %s %d", path, projectID))
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("XFS设置失败: %w", err)
	}
	return nil
}

// SetProjectIDExt4 在ext4文件系统上设置project ID
func SetProjectIDExt4(path string, projectID int) error {
	cmd := exec.Command("chattr", "+P", "-p", strconv.Itoa(projectID), path)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ext4设置失败: %w", err)
	}
	return nil
}

// SetProjectID 设置文件或目录的project ID
func SetProjectID(path string, projectID int) error {
	fsType, err := DetectFileSystem(path)
	if err != nil {
		return err
	}

	switch fsType {
	case "xfs":
		return SetProjectIDXFS(path, projectID)
	case "ext4":
		return SetProjectIDExt4(path, projectID)
	default:
		return fmt.Errorf("不支持的文件系统类型: %s", fsType)
	}
}

// GetProjectID 获取文件或目录的project ID
func GetProjectID(path string) (int, error) {
	cmd := exec.Command("lsattr", "-p", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("获取project ID失败: %w", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return 0, fmt.Errorf("未找到project ID")
	}

	parts := strings.Fields(output)
	for _, part := range parts {
		if strings.HasPrefix(part, "project_id=") {
			idStr := strings.TrimPrefix(part, "project_id=")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return 0, fmt.Errorf("解析project ID失败: %w", err)
			}
			return id, nil
		}
	}

	return 0, fmt.Errorf("未找到project ID")
}

// ClearProjectID 清除文件或目录的project ID
func ClearProjectID(path string) error {
	cmd := exec.Command("chattr", "-P", path)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("清除project ID失败: %w", err)
	}
	return nil
}
