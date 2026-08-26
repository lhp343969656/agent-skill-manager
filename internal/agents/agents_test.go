package agents

import (
	"context"
	"testing"
)

type mockAdapter struct {
	id    string
	name  string
	dirs  []Installation
	valid bool
}

func (m *mockAdapter) ID() string                        { return m.id }
func (m *mockAdapter) DisplayName() string               { return m.name }
func (m *mockAdapter) Detect(ctx context.Context) ([]Installation, error) {
	return m.dirs, nil
}
func (m *mockAdapter) ValidateSkillsDirectory(path string) error {
	if m.valid {
		return nil
	}
	return &InvalidSkillsDirError{Adapter: m.id, Reason: "mock invalid"}
}

func TestAdaptersIdentity(t *testing.T) {
	r := DefaultRegistry()

	codex, ok := r.Get("codex")
	if !ok {
		t.Fatal("缺少 codex 适配器")
	}
	if codex.DisplayName() != "Codex" {
		t.Errorf("codex 显示名错误: %s", codex.DisplayName())
	}

	opencode, ok := r.Get("opencode")
	if !ok {
		t.Fatal("缺少 opencode 适配器")
	}
	if opencode.DisplayName() != "OpenCode" {
		t.Errorf("opencode 显示名错误: %s", opencode.DisplayName())
	}
}

func TestRegistryListAndDetect(t *testing.T) {
	r := NewRegistry(
		&mockAdapter{id: "a", name: "A", dirs: []Installation{{SkillsPath: "/a", Detected: true}}},
		&mockAdapter{id: "b", name: "B", dirs: nil}, // b 未检测到
	)

	if len(r.List()) != 2 {
		t.Fatalf("期望 2 个适配器，实际 %d", len(r.List()))
	}

	result := r.DetectAll(context.Background())
	if _, ok := result["a"]; !ok {
		t.Error("a 应被检测到")
	}
	if _, ok := result["b"]; ok {
		t.Error("b 不应被检测到（无安装）")
	}
}

func TestValidateSkillsDirectory(t *testing.T) {
	r := NewRegistry(&mockAdapter{id: "a", name: "A", valid: false})
	a, _ := r.Get("a")
	if err := a.ValidateSkillsDirectory("/some/path"); err == nil {
		t.Error("期望校验失败，实际成功")
	}
}
