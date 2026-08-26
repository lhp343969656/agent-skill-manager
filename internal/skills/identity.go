package skills

import (
	"path"
	"strings"
)

// Identity 描述一个 Skill 的唯一身份（设计文档第 8 节）
type Identity struct {
	Owner       string
	Repository  string
	SkillPath   string // 相对路径（skill 所在目录）
}

// ID 返回 Skill 的唯一标识：github.com/{owner}/{repo}/{skill-path}
func (id Identity) ID() string {
	p := strings.Trim(id.SkillPath, "/")
	return path.Join("github.com", id.Owner, id.Repository, p)
}

// DisplayName 返回用于展示的名称
// 优先使用 Skill.md 中的 name；未解析前用路径最后一段
func (id Identity) DisplayName() string {
	p := strings.Trim(id.SkillPath, "/")
	if p == "" {
		return id.Repository
	}
	parts := strings.Split(p, "/")
	return parts[len(parts)-1]
}

// VersionDirName 返回不可变版本目录名（用 commit SHA）
func VersionDirName(commitSHA string) string {
	return commitSHA
}
