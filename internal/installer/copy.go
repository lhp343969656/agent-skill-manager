package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// filepathWalkCopy 递归复制目录 src 到 dst
func filepathWalkCopy(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 计算相对路径
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// 跳过符号链接，避免意外复制链接目标
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return copyOne(path, target)
	})
}

// copyOne 复制单个文件
func copyOne(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
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
		return fmt.Errorf("复制文件失败 %s -> %s: %w", src, dst, err)
	}
	return out.Close()
}
