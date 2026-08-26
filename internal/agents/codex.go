package agents

import (
	"context"
	"os"
	"path/filepath"
)

// CodexAdapter 适配 OpenAI Codex Agent
type CodexAdapter struct{}

// codexSkillsRelativePath 是 Codex Skill 目录相对于主目录的路径
// Codex 官方文档：用户级技能目录为 ~/.agents/skills（2025 年起统一，旧版 ~/.codex/skills 已废弃）
const codexSkillsRelativePath = ".agents/skills"

// ID 返回适配器标识
func (a *CodexAdapter) ID() string {
	return "codex"
}

// DisplayName 返回显示名称
func (a *CodexAdapter) DisplayName() string {
	return "Codex"
}

// Detect 检测 Codex 是否已安装及其 Skill 目录
func (a *CodexAdapter) Detect(ctx context.Context) ([]Installation, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	skillsDir := filepath.Join(home, codexSkillsRelativePath)
	// 检查 .codex 配置目录是否存在（判断是否安装）
	codexDir := filepath.Join(home, ".codex")
	if _, err := os.Stat(codexDir); os.IsNotExist(err) {
		return nil, ErrNotDetected
	}

	// 确保 skills 目录存在
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, err
	}

	return []Installation{{SkillsPath: skillsDir, Detected: true}}, nil
}

// ValidateSkillsDirectory 校验目录可用
func (a *CodexAdapter) ValidateSkillsDirectory(path string) error {
	if path == "" {
		return &InvalidSkillsDirError{Adapter: a.ID(), Reason: "目录不能为空"}
	}
	return os.MkdirAll(path, 0o755)
}
