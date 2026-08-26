package storage

import (
	"context"
	"path/filepath"
	"testing"

	"agent-skill-manager/internal/models"
)

func TestListSkillsReturnsDisplayVersion(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 插入 Skill 并写入版本记录，设置 currentVersionId
	skill := &models.Skill{
		ID:               "github.com/o/r",
		DisplayName:      "skill",
		SourceURL:        "https://github.com/o/r",
		RepositoryOwner:  "o",
		RepositoryName:   "r",
		CurrentVersionID: "abc123def",
	}
	if err := store.UpsertSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}

	version := &models.SkillVersion{
		ID:             "abc123def",
		SkillID:        skill.ID,
		DisplayVersion: "v1.2.3",
		GitRef:         "refs/tags/v1.2.3",
		CommitSHA:      "abc123def",
		InstallPath:    filepath.Join(dir, "pkg", "abc123def"),
	}
	if err := store.InsertVersion(ctx, version); err != nil {
		t.Fatal(err)
	}

	// 查询并验证 displayVersion
	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("期望 1 条，实际 %d", len(skills))
	}
	if skills[0].DisplayVersion != "v1.2.3" {
		t.Errorf("displayVersion 应为 v1.2.3，实际 %q", skills[0].DisplayVersion)
	}
	if skills[0].CurrentVersionID != "abc123def" {
		t.Errorf("currentVersionId 应为 abc123def，实际 %q", skills[0].CurrentVersionID)
	}
}

func TestListSkillsVersionEmptyForOldData(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 模拟老数据：Skill 存在但没有版本记录、currentVersionId 为空
	skill := &models.Skill{
		ID:               "local/product-prototype",
		DisplayName:      "product-prototype",
		SourceURL:        "H:/share-skills/local/product-prototype",
		RepositoryOwner:  "local",
		RepositoryName:   "product-prototype",
		CurrentVersionID: "",
	}
	if err := store.UpsertSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}

	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("期望 1 条，实际 %d", len(skills))
	}
	if skills[0].DisplayVersion != "" {
		t.Errorf("老数据 displayVersion 应为空，实际 %q", skills[0].DisplayVersion)
	}
}
