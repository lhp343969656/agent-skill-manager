package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLocalFile(t *testing.T) {
	// 构造一个本地 Skill 文件夹（含 SKILL.md）
	src := filepath.Join(t.TempDir(), "my-skill")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# My Local Skill"), 0o644)
	os.WriteFile(filepath.Join(src, "data.txt"), []byte("x"), 0o644)

	// 共享目录
	sharedDir := filepath.Join(t.TempDir(), "AgentSkills")
	inst := New(sharedDir)

	// 选择整个文件夹安装
	results, err := inst.InstallLocal(context.Background(), src)
	if err != nil {
		t.Fatalf("InstallLocal 失败: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("期望 1 个结果，实际 %d", len(results))
	}
	if results[0].SkillID != "local/my-skill" {
		t.Errorf("SkillID 错误: %s", results[0].SkillID)
	}

	// 验证已复制到共享目录
	installed := filepath.Join(sharedDir, "local", "my-skill")
	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md 未安装: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installed, "data.txt")); err != nil {
		t.Errorf("data.txt 未安装: %v", err)
	}
}

func TestInstallLocalOverwrite(t *testing.T) {
	src := filepath.Join(t.TempDir(), "skill-a")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("v1"), 0o644)

	sharedDir := filepath.Join(t.TempDir(), "AgentSkills")
	inst := New(sharedDir)

	// 第一次安装
	if _, err := inst.InstallLocal(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	// 修改源文件为 v2
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("v2"), 0o644)

	// 覆盖安装
	if _, err := inst.InstallLocal(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	// 验证是 v2（覆盖更新）
	data, err := os.ReadFile(filepath.Join(sharedDir, "local", "skill-a", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v2" {
		t.Errorf("覆盖更新失败，内容为 %q", string(data))
	}
}

func TestInstallLocalNonSkillFile(t *testing.T) {
	// 选择非 SKILL.md 文件应报错
	badFile := filepath.Join(t.TempDir(), "readme.txt")
	os.WriteFile(badFile, []byte("hi"), 0o644)

	sharedDir := filepath.Join(t.TempDir(), "AgentSkills")
	inst := New(sharedDir)
	if _, err := inst.InstallLocal(context.Background(), badFile); err == nil {
		t.Fatal("选择非 SKILL.md 文件应报错")
	}
}
