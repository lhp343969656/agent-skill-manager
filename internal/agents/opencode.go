package agents

import (
	"context"
	"os"
	"path/filepath"
)

// OpenCodeAdapter 适配 OpenCode Agent
type OpenCodeAdapter struct{}

// opencodeSkillsRelativePath 是 OpenCode Skill 目录相对于主目录的路径
const opencodeSkillsRelativePath = ".config/opencode/skill"

// ID 返回适配器标识
func (a *OpenCodeAdapter) ID() string {
	return "opencode"
}

// DisplayName 返回显示名称
func (a *OpenCodeAdapter) DisplayName() string {
	return "OpenCode"
}

// Detect 检测 OpenCode 是否已安装及其 Skill 目录
func (a *OpenCodeAdapter) Detect(ctx context.Context) ([]Installation, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	skillsDir := filepath.Join(home, opencodeSkillsRelativePath)
	// 检查 opencode 配置目录是否存在
	opencodeDir := filepath.Join(home, ".config", "opencode")
	if _, err := os.Stat(opencodeDir); os.IsNotExist(err) {
		return nil, ErrNotDetected
	}

	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, err
	}

	return []Installation{{SkillsPath: skillsDir, Detected: true}}, nil
}

// ValidateSkillsDirectory 校验目录可用
func (a *OpenCodeAdapter) ValidateSkillsDirectory(path string) error {
	if path == "" {
		return &InvalidSkillsDirError{Adapter: a.ID(), Reason: "目录不能为空"}
	}
	return os.MkdirAll(path, 0o755)
}
