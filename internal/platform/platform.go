package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// targetIsNonEmpty 判断 target 是否存在且非空（用于冲突检测）。
// 空目录不算冲突，可以填充内容。
func targetIsNonEmpty(target string) (bool, error) {
	info, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// 符号链接本身即视为"有内容"，避免重复覆盖
		return true, nil
	}

	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			return false, err
		}
		return len(entries) > 0, nil
	}

	// 普通文件
	return true, nil
}

// copyManaged 执行托管复制：把 source 目录/文件完整复制到 target。
// 若 target 已存在非空内容，返回 ConflictError。
func copyManaged(source, target string) error {
	nonEmpty, err := targetIsNonEmpty(target)
	if err != nil {
		return err
	}
	if nonEmpty {
		return &ConflictError{Target: target}
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("无法读取来源: %w", err)
	}

	if info.IsDir() {
		if err := copyDir(source, target); err != nil {
			return err
		}
	} else {
		if err := copyFile(source, target); err != nil {
			return err
		}
	}
	return nil
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else if entry.Type()&os.ModeSymlink == 0 {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// removeFileOrEmptyDir 删除一个文件，或仅删除空目录。
// 如果目录非空（可能混入用户内容），不删除。
func removeFileOrEmptyDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			// 目录非空，可能是用户后来又添加了内容，不删除
			return fmt.Errorf("目录非空，拒绝删除: %s", path)
		}
		return os.Remove(path)
	}
	return os.Remove(path)
}
