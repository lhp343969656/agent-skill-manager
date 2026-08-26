//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WindowsLinker 实现 Windows 平台的链接策略
// 优先级：Junction -> Symlink -> 托管复制
type WindowsLinker struct{}

// NewLinker 创建当前平台的链接器
func NewLinker() Linker {
	return &WindowsLinker{}
}

// Create 创建链接，按 Junction -> Symlink -> Copy 顺序尝试
func (l *WindowsLinker) Create(source, target string) (LinkMode, error) {
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

	// 冲突检测：target 已存在非空内容则不覆盖
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

	// 判断 source 是否为目录
	info, err := os.Stat(absSource)
	if err != nil {
		return "", fmt.Errorf("无法读取来源: %w", err)
	}

	var lastErr error
	// 1. 目录使用 Junction（无需管理员权限）
	if info.IsDir() {
		if err := createJunction(absSource, absTarget); err == nil {
			return LinkModeJunction, nil
		} else {
			lastErr = err
		}
	}

	// 2. 尝试 Symlink（Windows 需开发者模式或管理员）
	if err := os.Symlink(absSource, absTarget); err == nil {
		return LinkModeSymlink, nil
	} else {
		lastErr = err
	}

	// 不再使用复制兜底：建不了链接直接返回错误，避免出现难以管理的复制内容
	return "", fmt.Errorf("无法创建链接（目录联接和符号链接均失败），且已禁用复制兜底：%v", lastErr)
}

// Inspect 检查 target 的链接状态
func (l *WindowsLinker) Inspect(target string) (LinkInfo, error) {
	return inspectCommon(target)
}

// RemoveManaged 移除由管理器创建的链接，删除前校验目标指向 expectedSource
func (l *WindowsLinker) RemoveManaged(target, expectedSource string) error {
	return removeManagedCommon(target, expectedSource)
}

// createJunction 使用 cmd 的 mklink /J 命令创建目录联接（无需管理员权限）
func createJunction(source, target string) error {
	// 确保目标路径不存在（mklink 要求目标不存在）
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command("cmd", "/C", "mklink", "/J", target, source)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建 Junction 失败: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
