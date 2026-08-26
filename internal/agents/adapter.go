package agents

import (
	"context"
	"errors"
)

// Installation 描述检测到的 Agent 安装位置
type Installation struct {
	// SkillsPath 是 Agent 存放 Skill 的目录
	SkillsPath string
	// Detected 表示是否是自动检测到的（否则为用户手动配置）
	Detected bool
}

// AgentAdapter 是统一 Agent 适配器接口（设计文档第 9 节）
type AgentAdapter interface {
	// ID 返回适配器的唯一标识，如 "codex"、"opencode"
	ID() string
	// DisplayName 返回 Agent 的显示名称
	DisplayName() string
	// Detect 自动检测 Agent 的默认安装位置和 Skill 目录
	Detect(ctx context.Context) ([]Installation, error)
	// ValidateSkillsDirectory 校验用户的 Skill 目录是否有效
	ValidateSkillsDirectory(path string) error
}

// ErrNotDetected 表示未在系统中检测到该 Agent
var ErrNotDetected = errors.New("未检测到该 Agent")
