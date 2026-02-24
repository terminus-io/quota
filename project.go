package quota

import (
	"fmt"
)

// GetFilesystemType 获取指定路径的文件系统类型
func GetFilesystemType(path string) (FileSystemType, error) {
	return DetectFileSystem(path)
}

// SetProjectIDXFS 在XFS文件系统上设置project ID（使用ioctl系统调用）
func SetProjectIDXFS(path string, projectID int) error {
	return setProjectIDXFS(path, projectID)
}

// SetProjectIDExt4 在ext4文件系统上设置project ID（使用ioctl系统调用）
func SetProjectIDExt4(path string, projectID int) error {
	return setProjectIDExt4(path, projectID)
}

// SetProjectID 设置文件或目录的project ID（使用ioctl系统调用）
func SetProjectID(path string, projectID int) error {
	fsType, err := DetectFileSystem(path)
	if err != nil {
		return err
	}

	switch fsType {
	case FileSystemXFS:
		return setProjectIDXFS(path, projectID)
	case FileSystemEXT4:
		return setProjectIDExt4(path, projectID)
	default:
		return fmt.Errorf("不支持的文件系统类型: %s", fsType)
	}
}

// GetProjectID 获取文件或目录的project ID（使用ioctl系统调用）
func GetProjectID(path string) (int, error) {
	return getProjectID(path)
}

// ClearProjectID 清除文件或目录的project ID（使用ioctl系统调用）
func ClearProjectID(path string) error {
	return clearProjectID(path)
}
