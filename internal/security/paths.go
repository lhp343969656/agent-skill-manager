package security

import (
	"path/filepath"
	"strings"
)

// SanitizeEntryPath 校验 zip 中的条目路径，防止 Zip Slip（路径穿越）。
// 返回安全的目标路径；若条目非法则返回错误。
func SanitizeEntryPath(baseDir, entryName string) (string, error) {
	// 拒绝绝对路径条目（Windows 盘符 / Unix 根路径）
	clean := filepath.ToSlash(entryName)
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") ||
		(len(clean) >= 2 && clean[1] == ':') {
		return "", &ZipSlipError{Entry: entryName}
	}
	clean = strings.TrimPrefix(clean, "/")

	// 用 baseDir 拼接后检查是否越界
	target := filepath.Join(baseDir, filepath.FromSlash(clean))

	// 校验 target 必须在 baseDir 之内
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &ZipSlipError{Entry: entryName}
	}

	return target, nil
}

// ZipSlipError 表示检测到路径穿越攻击
type ZipSlipError struct {
	Entry string
}

func (e *ZipSlipError) Error() string {
	return "检测到非法压缩包路径（Zip Slip）: " + e.Entry
}

// ValidateName 校验解压后的顶层条目名是否安全（用于定位内容根目录）
func ValidateName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.Contains(name, "..")
}
