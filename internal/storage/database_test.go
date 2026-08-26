package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-skill-manager/internal/models"
)

func TestOpenCreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	// 验证 .manager 目录和 state.db 已创建
	stateDB := filepath.Join(dir, ".manager", "state.db")
	if _, err := os.Stat(stateDB); err != nil {
		t.Fatalf("state.db 未创建: %v", err)
	}

	// 验证核心表存在
	tables := []string{"skills", "skill_versions", "agents", "managed_links"}
	for _, name := range tables {
		var cnt int
		err := store.DB().QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name,
		).Scan(&cnt)
		if err != nil {
			t.Fatalf("查询表 %s 失败: %v", name, err)
		}
		if cnt == 0 {
			t.Errorf("表 %s 未创建", name)
		}
	}
}

func TestSkillCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	sk := &models.Skill{
		ID:               "github.com/owner/repo/path",
		DisplayName:      "Test Skill",
		SourceURL:        "https://github.com/owner/repo",
		RepositoryOwner:  "owner",
		RepositoryName:   "repo",
		RepositoryPath:   "path",
		CurrentVersionID: "",
	}
	if err := store.InsertSkill(ctx, sk); err != nil {
		t.Fatalf("InsertSkill 失败: %v", err)
	}

	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatalf("ListSkills 失败: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("期望 1 条 Skill，实际 %d", len(skills))
	}
	if skills[0].DisplayName != "Test Skill" {
		t.Errorf("DisplayName 不正确: %s", skills[0].DisplayName)
	}
}

func TestVersionAndLink(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	sk := &models.Skill{
		ID:              "github.com/o/r/p",
		DisplayName:     "Skill",
		SourceURL:       "https://github.com/o/r",
		RepositoryOwner: "o",
		RepositoryName:  "r",
		RepositoryPath:  "p",
	}
	if err := store.InsertSkill(ctx, sk); err != nil {
		t.Fatal(err)
	}

	v := &models.SkillVersion{
		ID:             "v1",
		SkillID:        sk.ID,
		DisplayVersion: "1.0.0",
		GitRef:         "refs/tags/v1.0.0",
		CommitSHA:      "abc123",
		Checksum:       "sha256:xyz",
		InstallPath:    filepath.Join(dir, "pkg", "v1"),
		InstalledAt:    time.Now(),
	}
	if err := store.InsertVersion(ctx, v); err != nil {
		t.Fatalf("InsertVersion 失败: %v", err)
	}

	agent := &models.Agent{
		ID:          "codex",
		AdapterID:   "codex",
		DisplayName: "Codex",
		SkillsPath:  filepath.Join(dir, "codex-skills"),
		Enabled:     true,
		Detected:    true,
	}
	if err := store.UpsertAgent(ctx, agent); err != nil {
		t.Fatalf("UpsertAgent 失败: %v", err)
	}

	agents, err := store.ListAgents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || !agents[0].Enabled {
		t.Errorf("Agent 保存不正确: %+v", agents)
	}

	link := &models.ManagedLink{
		ID:             "l1",
		AgentID:        agent.ID,
		SkillID:        sk.ID,
		SkillVersionID: v.ID,
		SourcePath:     v.InstallPath,
		TargetPath:     filepath.Join(agent.SkillsPath, "Skill"),
		LinkMode:       "symlink",
		Status:         "active",
	}
	if err := store.InsertLink(ctx, link); err != nil {
		t.Fatalf("InsertLink 失败: %v", err)
	}
}

func TestUpsertSkill(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 首次插入
	sk := &models.Skill{
		ID:          "local/my-skill",
		DisplayName: "my-skill",
		SourceURL:   "D:/Skills/my-skill",
	}
	if err := store.UpsertSkill(ctx, sk); err != nil {
		t.Fatalf("UpsertSkill 首次插入失败: %v", err)
	}

	// 覆盖更新（同一 ID，改显示名和来源）
	sk.DisplayName = "my-skill-v2"
	sk.SourceURL = "E:/Skills/my-skill"
	if err := store.UpsertSkill(ctx, sk); err != nil {
		t.Fatalf("UpsertSkill 覆盖更新失败: %v", err)
	}

	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("覆盖更新后应为 1 条记录，实际 %d", len(skills))
	}
	if skills[0].DisplayName != "my-skill-v2" {
		t.Errorf("覆盖更新未生效，DisplayName 为 %s", skills[0].DisplayName)
	}
	if skills[0].SourceURL != "E:/Skills/my-skill" {
		t.Errorf("覆盖更新未更新 SourceURL: %s", skills[0].SourceURL)
	}
}
