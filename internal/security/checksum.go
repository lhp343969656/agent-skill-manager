package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SHA256 计算文件或目录的 SHA-256 校验和。
// 目录会递归计算所有文件内容的组合哈希。
func SHA256(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return sha256File(path)
	}
	return sha256Dir(path)
}

// sha256File 计算单个文件的 SHA-256
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sha256Dir 递归计算目录下所有文件的哈希组合
func sha256Dir(dir string) (string, error) {
	h := sha256.New()
	err := walkAndHash(dir, h)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// walkAndHash 遍历目录，把每个文件的相对路径和内容哈希喂给 sha256
func walkAndHash(root string, h io.Writer) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fh, err := sha256File(path)
		if err != nil {
			return err
		}
		// 写入相对路径 + 哈希，保证顺序敏感性
		fmt.Fprintf(h, "%s|%s\n", path, fh)
		return nil
	})
}

// Verify 校验文件/目录的哈希是否匹配
func Verify(path, expected string) (bool, error) {
	actual, err := SHA256(path)
	if err != nil {
		return false, err
	}
	return actual == expected, nil
}
