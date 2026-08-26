package skills

import (
	"regexp"
	"strings"
)

// SkillInfo 是验证并解析后的 Skill 信息
type SkillInfo struct {
	Name        string
	Description string
	Path        string // SKILL.md 绝对路径
}

// yamlFrontMatter 匹配 YAML front-matter 块（--- 开头）
var frontMatterRe = regexp.MustCompile(`(?s)^\s*---\s*\n(.*?)\n---`)

// keyRe 匹配 front-matter 中的 key: value
var keyRe = regexp.MustCompile(`(?m)^\s*([a-zA-Z][a-zA-Z0-9_-]*)\s*:\s*(.+?)\s*$`)

// Validator 验证 SKILL.md 格式
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// Validate 验证 SKILL.md 是否有效，并解析元数据。
// 宽松规则：文件有内容即可视为有效 Skill；有 YAML front-matter 时解析 name/description。
func (v *Validator) Validate(file SkillFile) (SkillInfo, error) {
	info := SkillInfo{Path: file.Path}

	content := strings.TrimSpace(file.Content)
	if content == "" {
		return info, &InvalidSkillError{Path: file.Path, Reason: "SKILL.md 内容为空"}
	}

	// 解析 front-matter
	info.Name = ""
	info.Description = ""
	if m := frontMatterRe.FindStringSubmatch(content); len(m) == 2 {
		for _, kv := range keyRe.FindAllStringSubmatch(m[1], -1) {
			switch strings.ToLower(kv[1]) {
			case "name":
				info.Name = strings.TrimSpace(kv[2])
			case "description":
				info.Description = strings.TrimSpace(kv[2])
			}
		}
	}

	return info, nil
}

// InvalidSkillError 表示 Skill 文件无效
type InvalidSkillError struct {
	Path   string
	Reason string
}

func (e *InvalidSkillError) Error() string {
	return "无效的 Skill: " + e.Path + " (" + e.Reason + ")"
}
