package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkerCreateAndInspect(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	// 在 source 里放一个文件
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "linked")

	linker := NewLinker()
	mode, err := linker.Create(source, target)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if mode == "" {
		t.Fatal("Create 返回空 mode")
	}

	// Inspect 验证
	info, err := linker.Inspect(target)
	if err != nil {
		t.Fatalf("Inspect 失败: %v", err)
	}
	if !info.Exists {
		t.Errorf("Inspect 期望存在链接，实际不存在")
	}
	if info.SourcePath == "" {
		t.Errorf("Inspect 未能解析来源路径")
	}

	// 验证通过链接能读到文件
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Errorf("通过链接访问文件失败: %v", err)
	}
}

func TestLinkerConflict(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "existing")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	// 目标已存在非空内容（用户内容）
	if err := os.WriteFile(filepath.Join(target, "user-file.txt"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}

	linker := NewLinker()
	_, err := linker.Create(source, target)
	if err == nil {
		t.Fatal("期望冲突错误，实际没有")
	}
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("期望 ConflictError，实际: %v", err)
	}

	// 验证用户内容未被覆盖
	data, err := os.ReadFile(filepath.Join(target, "user-file.txt"))
	if err != nil || string(data) != "user" {
		t.Errorf("冲突时用户内容被覆盖或损坏")
	}
}

func TestLinkerRemoveManaged(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "linked")

	linker := NewLinker()
	if _, err := linker.Create(source, target); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 正确删除
	if err := linker.RemoveManaged(target, source); err != nil {
		t.Fatalf("RemoveManaged 失败: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("目标应已删除，实际: %v", err)
	}

	// 来源应保留
	if _, err := os.Stat(source); err != nil {
		t.Errorf("来源不应被删除: %v", err)
	}
}

func TestRemoveManagedMismatch(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "linked")

	linker := NewLinker()
	if _, err := linker.Create(source, target); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 用错误的 expectedSource 删除，应报错且不删除
	err := linker.RemoveManaged(target, other)
	if err == nil {
		t.Fatal("期望链接不匹配错误，实际没有")
	}
	if _, err := os.Lstat(target); err != nil {
		t.Errorf("不匹配时不应删除目标，实际: %v", err)
	}
}
