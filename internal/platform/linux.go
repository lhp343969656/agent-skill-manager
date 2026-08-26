//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// LinuxLinker 实现 Linux 平台的链接策略
// 优先级：Symlink（Linux 原生支持）
type LinuxLinker struct{}

// NewLinker 创建当前平台的链接器
func NewLinker() Linker {
	return &LinuxLinker{}
}

// Create 创建链接，使用 Symlink
func (l *LinuxLinker) Create(source, target string) (LinkMode, error) {
	if source == "" || target == "" {
		return "", fmt.Errorf("source 和 target 不能为空")
	}
	absSource, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	// 冲突检测
	nonEmpty, err := targetIsNonEmpty(absTarget)
	if err != nil {
		return "", err
	}
	if nonEmpty {
		return "", &ConflictError{Target: absTarget}
	}

	// 确保父目录存在
	parent := filepath.Dir(absTarget)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}

	// 1. 尝试 Symlink
	if err := os.Symlink(absSource, absTarget); err != nil {
		// 不再使用复制兜底：建不了链接直接返回错误，避免出现难以管理的复制内容
		return "", fmt.Errorf("无法创建符号链接，且已禁用复制兜底：%v", err)
	}
	return LinkModeSymlink, nil
}

// Inspect 检查 target 的链接状态
func (l *LinuxLinker) Inspect(target string) (LinkInfo, error) {
	return inspectCommon(target)
}

// RemoveManaged 移除由管理器创建的链接
func (l *LinuxLinker) RemoveManaged(target, expectedSource string) error {
	return removeManagedCommon(target, expectedSource)
}