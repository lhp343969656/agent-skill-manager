package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerFindsSKILL(t *testing.T) {
	dir := t.TempDir()
	// 构造多层目录，含多个 SKILL.md
	os.MkdirAll(filepath.Join(dir, "skills", "a"), 0o755)
	os.MkdirAll(filepath.Join(dir, "skills", "b"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(dir, "skills", "a", "SKILL.md"), []byte("# A"), 0o644)
	os.WriteFile(filepath.Join(dir, "skills", "b", "SKILL.md"), []byte("# B"), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden", "SKILL.md"), []byte("# hidden"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644) // 非 SKILL.md 不应匹配

	s := NewScanner()
	files, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("期望找到 2 个 SKILL.md，实际 %d", len(files))
	}
}

func TestValidatorParsesFrontMatter(t *testing.T) {
	file := SkillFile{
		Path:    "/tmp/x/SKILL.md",
		Content: "---\nname: My Skill\ndescription: Does great things\n---\n# Body",
	}
	v := NewValidator()
	info, err := v.Validate(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "My Skill" {
		t.Errorf("name 解析不正确: %q", info.Name)
	}
	if info.Description != "Does great things" {
		t.Errorf("description 解析不正确: %q", info.Description)
	}
}

func TestValidatorRejectsEmpty(t *testing.T) {
	file := SkillFile{Path: "/tmp/x/SKILL.md", Content: "   "}
	v := NewValidator()
	_, err := v.Validate(file)
	if err == nil {
		t.Fatal("期望空内容报错")
	}
	if _, ok := err.(*InvalidSkillError); !ok {
		t.Errorf("期望 InvalidSkillError，实际 %T", err)
	}
}

func TestValidatorAcceptPlainMarkdown(t *testing.T) {
	// 无 front-matter 的纯 markdown 也应视为有效
	file := SkillFile{Path: "/tmp/x/SKILL.md", Content: "# Just a title\n\nSome instructions."}
	v := NewValidator()
	info, err := v.Validate(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "" {
		t.Errorf("无 front-matter 时 name 应为空，实际 %q", info.Name)
	}
}
