package agents

import (
	"context"
	"fmt"
	"path/filepath"

	"agent-skill-manager/internal/platform"
)

// Syncer 负责将共享 Skill 同步到已启用的 Agent
type Syncer struct {
	linker platform.Linker
	// agents 是已启用的 Agent（adapterId -> skills 目录）
	agents map[string]string
}

// NewSyncer 创建同步器
func NewSyncer(linker platform.Linker, agents map[string]string) *Syncer {
	return &Syncer{linker: linker, agents: agents}
}

// SyncResult 描述一次同步的结果
type SyncResult struct {
	AgentID   string
	SkillName string // Agent 目录中的 Skill 名（目录名）
	Status    string // linked | conflict | skipped
	Detail    string
}

// SyncSkills 将 source 目录中的每个 Skill 同步到所有已启用 Agent。
// sourceMap 是 skillID -> 版本目录路径 的映射。
// 返回每个 Agent 的处理结果。
func (s *Syncer) SyncSkills(ctx context.Context, sourceMap map[string]string) ([]SyncResult, error) {
	var results []SyncResult

	for agentID, skillsDir := range s.agents {
		for skillName, versionDir := range sourceMap {
			target := filepath.Join(skillsDir, skillName)

			mode, err := s.linker.Create(versionDir, target)
			if err != nil {
				// 冲突：跳过并记录
				var conflict *platform.ConflictError
				if asConflict(err, &conflict) {
					results = append(results, SyncResult{
						AgentID:   agentID,
						SkillName: skillName,
						Status:    "conflict",
						Detail:    "目标位置已存在同名内容",
					})
					continue
				}
				// 其他错误：记录为 skipped
				results = append(results, SyncResult{
					AgentID:   agentID,
					SkillName: skillName,
					Status:    "skipped",
					Detail:    err.Error(),
				})
				continue
			}

			results = append(results, SyncResult{
				AgentID:   agentID,
				SkillName: skillName,
				Status:    "linked",
				Detail:    fmt.Sprintf("链接模式: %s", mode),
			})
		}
	}

	return results, nil
}

// asConflict 判断错误是否为 ConflictError
func asConflict(err error, target **platform.ConflictError) bool {
	ce, ok := err.(*platform.ConflictError)
	if !ok {
		return false
	}
	*target = ce
	return true
}
