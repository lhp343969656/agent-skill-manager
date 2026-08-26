package platform

import (
	"os"
	"path/filepath"
)

// inspectCommon 检查 target 路径的链接状态（跨平台共用）
func inspectCommon(target string) (LinkInfo, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return LinkInfo{}, err
	}

	info := LinkInfo{}

	// 目标本身是否存在
	_, err = os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil // 不存在，Exists=false
		}
		return LinkInfo{}, err
	}
	info.TargetExists = true

	// 符号链接
	fi, err := os.Lstat(abs)
	if err != nil {
		return LinkInfo{}, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		info.Exists = true
		info.Mode = LinkModeSymlink
		info.SourcePath = readLinkTarget(abs)
		return info, nil
	}

	// junction（Windows 重解析点）
	if isJunction(abs) {
		info.Exists = true
		info.Mode = LinkModeJunction
		info.SourcePath = readLinkTarget(abs)
		return info, nil
	}

	// 普通文件/目录（非链接），无法确定是否为托管复制，仅标记存在
	info.Exists = true
	return info, nil
}

// isJunction 判断路径是否为 Junction。
// 在 macOS 上始终返回 false；在 Windows 上通过 junction 特性判断。
func isJunction(path string) bool {
	return isJunctionPlatform(path)
}

// readLinkTarget 读取链接（symlink 或 junction）指向的目标路径。
// os.Readlink 对 Windows junction 也能正确解析（EvalSymlinks 对 junction 不可靠）。
func readLinkTarget(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return target
}

// removeManagedCommon 移除管理器创建的链接：
// 1. 确认 target 存在且（若是链接）指向 expectedSource
// 2. 若是链接，删除链接本身（不删 source）
// 3. 若是托管复制，仅删除该目录/文件，且目录非空时拒绝
func removeManagedCommon(target, expectedSource string) error {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	_, err = os.Lstat(absTarget)
	if err != nil {
		if os.IsNotExist(err) {
			// 目标已不存在，视为删除成功
			return nil
		}
		return err
	}

	// 若是 symlink，校验其指向是否与 expectedSource 匹配
	fi, err := os.Lstat(absTarget)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		resolved := readLinkTarget(absTarget)
		if !samePath(resolved, expectedSource) {
			return &LinkMismatchError{Target: absTarget, Got: resolved, Expected: expectedSource}
		}
		return os.Remove(absTarget)
	}

	// junction：校验指向后删除
	if isJunction(absTarget) {
		resolved := readLinkTarget(absTarget)
		if !samePath(resolved, expectedSource) {
			return &LinkMismatchError{Target: absTarget, Got: resolved, Expected: expectedSource}
		}
		return os.RemoveAll(absTarget)
	}

	// 托管复制：删除文件或空目录；非空目录拒绝
	return removeFileOrEmptyDir(absTarget)
}

// LinkMismatchError 表示链接目标与预期来源不匹配
type LinkMismatchError struct {
	Target   string
	Got      string
	Expected string
}

func (e *LinkMismatchError) Error() string {
	return "链接指向与预期不符，拒绝删除（可能是用户改动过）: " + e.Target
}

// samePath 比较两个路径是否指向同一位置（大小写不敏感，消除 .. 与 \\?\ 前缀）
func samePath(a, b string) bool {
	aa := normalizeForCompare(a)
	bb := normalizeForCompare(b)
	return aa == bb
}

func normalizeForCompare(p string) string {
	// 去掉 Windows 的 \\?\ 前缀
	if len(p) >= 4 && (p[0] == '\\' && p[1] == '\\' && p[2] == '?' && p[3] == '\\') {
		p = p[4:]
	}
	p = filepath.Clean(p)
	if os.PathSeparator == '\\' {
		return toLower(p)
	}
	return p
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
