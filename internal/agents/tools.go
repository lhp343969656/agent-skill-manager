package agents

import (
	"context"
	"os"
	"path/filepath"
)

// simpleAgentAdapter 适用于通过 ~/.<tool>/skills 约定存放技能的常见 AI 编程工具。
// 这些工具的安装判断与技能目录结构一致：配置文件目录的 skills 子目录即为技能目录。
type simpleAgentAdapter struct {
	id    string
	disp  string
	// configRelPath 相对用户主目录的配置目录（用于判断是否已安装）
	configRelPath string
	// skillsRelPath 相对用户主目录的技能目录
	skillsRelPath string
}

func (a *simpleAgentAdapter) ID() string          { return a.id }
func (a *simpleAgentAdapter) DisplayName() string { return a.disp }

// Detect 检测 Agent 是否已安装及其 Skill 目录。
// 若配置目录不存在则视为未安装。
func (a *simpleAgentAdapter) Detect(ctx context.Context) ([]Installation, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(home, a.configRelPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, ErrNotDetected
	}
	skillsDir := filepath.Join(home, a.skillsRelPath)
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, err
	}
	return []Installation{{SkillsPath: skillsDir, Detected: true}}, nil
}

// ValidateSkillsDirectory 校验目录可用
func (a *simpleAgentAdapter) ValidateSkillsDirectory(path string) error {
	if path == "" {
		return &InvalidSkillsDirError{Adapter: a.ID(), Reason: "目录不能为空"}
	}
	return os.MkdirAll(path, 0o755)
}

// newSimpleAgentAdapter 创建简单适配器，配置目录取技能目录的父目录。
func newSimpleAgentAdapter(id, displayName, skillsRelPath string) *simpleAgentAdapter {
	return &simpleAgentAdapter{
		id:            id,
		disp:          displayName,
		configRelPath: filepath.Dir(skillsRelPath),
		skillsRelPath: skillsRelPath,
	}
}

// buildAdditionalAdapters 返回 Codex / OpenCode 之外、基于 skill 目录约定的常见 AI 编程工具适配器。
// 技能目录参考各工具官方文档确认（均在用户主目录的隐藏配置目录下）。
func buildAdditionalAdapters() []AgentAdapter {
	return []AgentAdapter{
		newSimpleAgentAdapter("workbuddy", "WorkBuddy", ".workbuddy/skills"),
		newSimpleAgentAdapter("codebuddy", "CodeBuddy", ".codebuddy/skills"),
		newSimpleAgentAdapter("zcode", "ZCode", ".zcode/skills"),
		newSimpleAgentAdapter("deepseekharness", "DeepSeek Harness", ".dsh/skills"),
		newSimpleAgentAdapter("trae", "Trae", ".trae/skills"),
		newSimpleAgentAdapter("trae-work", "Trae Work", ".trae-cn/skills"),
		newSimpleAgentAdapter("qoder", "Qoder", ".qoder/skills"),
		newSimpleAgentAdapter("reasonix", "Reasonix", ".reasonix/skills"),
		newSimpleAgentAdapter("hermes", "Hermes", ".hermes/skills"),
	}
}
