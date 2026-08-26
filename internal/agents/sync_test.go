package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-skill-manager/internal/platform"
)

func TestSyncerLinksSkills(t *testing.T) {
	dir := t.TempDir()

	// 构造一个 Skill 版本目录
	versionDir := filepath.Join(dir, "version")
	os.MkdirAll(versionDir, 0o755)
	os.WriteFile(filepath.Join(versionDir, "SKILL.md"), []byte("# Skill"), 0o644)

	// 构造一个 Agent skills 目录
	agentDir := filepath.Join(dir, "agent-skills")
	os.MkdirAll(agentDir, 0o755)

	syncer := NewSyncer(platform.NewLinker(), map[string]string{
		"codex": agentDir,
	})

	results, err := syncer.SyncSkills(context.Background(), map[string]string{
		"my-skill": versionDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("期望 1 个结果，实际 %d", len(results))
	}
	if results[0].Status != "linked" {
		t.Fatalf("期望 linked，实际 %s (%s)", results[0].Status, results[0].Detail)
	}

	// 验证链接生效：Agent 目录下能访问到 SKILL.md
	if _, err := os.Stat(filepath.Join(agentDir, "my-skill", "SKILL.md")); err != nil {
		t.Errorf("Agent 目录无法访问 Skill: %v", err)
	}
}

func TestSyncerConflict(t *testing.T) {
	dir := t.TempDir()

	versionDir := filepath.Join(dir, "version")
	os.MkdirAll(versionDir, 0o755)
	os.WriteFile(filepath.Join(versionDir, "SKILL.md"), []byte("# Skill"), 0o644)

	// Agent 目录中已有同名内容（用户内容）
	agentDir := filepath.Join(dir, "agent-skills")
	os.MkdirAll(filepath.Join(agentDir, "my-skill"), 0o755)
	os.WriteFile(filepath.Join(agentDir, "my-skill", "user.md"), []byte("user"), 0o644)

	syncer := NewSyncer(platform.NewLinker(), map[string]string{
		"codex": agentDir,
	})

	results, err := syncer.SyncSkills(context.Background(), map[string]string{
		"my-skill": versionDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != "conflict" {
		t.Fatalf("期望 conflict，实际 %s", results[0].Status)
	}

	// 验证用户内容未被覆盖
	data, _ := os.ReadFile(filepath.Join(agentDir, "my-skill", "user.md"))
	if string(data) != "user" {
		t.Error("冲突时用户内容被覆盖")
	}
}
