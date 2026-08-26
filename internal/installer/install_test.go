package installer

import (
	"os"
	"path/filepath"
	"testing"

	"agent-skill-manager/internal/github"
)

func TestBuildSkillID(t *testing.T) {
	// 仓库根 + 技能名
	ref := &github.RepoRef{Owner: "owner", Repo: "repo", SubPath: ""}
	if id := buildSkillID(ref, "brainstorming"); id != "github.com/owner/repo/brainstorming" {
		t.Errorf("仓库根 Skill ID 错误: %s", id)
	}
	// 无技能名（保留旧行为）
	if id := buildSkillID(ref, ""); id != "github.com/owner/repo" {
		t.Errorf("无技能名 Skill ID 错误: %s", id)
	}

	// 子目录 + 技能名
	ref2 := &github.RepoRef{Owner: "owner", Repo: "repo", SubPath: "skills/abc"}
	if id := buildSkillID(ref2, "my-skill"); id != "github.com/owner/repo/skills/abc/my-skill" {
		t.Errorf("子目录 Skill ID 错误: %s", id)
	}
}

func TestCopySkillDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# Skill"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "data.txt"), []byte("data"), 0o644)

	dst := filepath.Join(t.TempDir(), "dst")
	if err := filepathWalkCopy(src, dst); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md 未复制: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "data.txt")); err != nil {
		t.Errorf("子目录文件未复制: %v", err)
	}
}

func TestVersionDirStructure(t *testing.T) {
	shared := filepath.Join(t.TempDir(), "AgentSkills")
	commit := "abc123def"

	// 验证 packagesRoot 和版本目录结构
	pkgRoot := packagesRoot(shared)
	versionDir := filepath.Join(pkgRoot, skillPath("github.com/o/r"), commitDir(commit))

	want := filepath.Join(shared, "packages", "github.com", "o", "r", "abc123def")
	if versionDir != want {
		t.Errorf("版本目录路径错误:\n got %s\nwant %s", versionDir, want)
	}
}
