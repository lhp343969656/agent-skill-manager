package config

import (
	"path/filepath"
	"testing"
)

// 由于 configDirectory 依赖系统环境变量（LOCALAPPDATA / 用户主目录），
// 这里通过构造 Manager 直接测试配置的读写持久化。
func TestConfigPersist(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{
		configDir:  dir,
		configFile: filepath.Join(dir, "config.json"),
	}

	if err := m.Load(); err != nil {
		t.Fatalf("加载空配置失败: %v", err)
	}

	if err := m.SetSharedDir("D:\\AgentSkills"); err != nil {
		t.Fatalf("设置共享目录失败: %v", err)
	}
	if m.GetConfig().SharedDir != "D:\\AgentSkills" {
		t.Errorf("共享目录未正确保存: %s", m.GetConfig().SharedDir)
	}

	// 重新加载验证持久化
	m2 := &Manager{
		configDir:  dir,
		configFile: filepath.Join(dir, "config.json"),
	}
	if err := m2.Load(); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if m2.GetConfig().SharedDir != "D:\\AgentSkills" {
		t.Errorf("持久化失败，重新加载后共享目录为: %s", m2.GetConfig().SharedDir)
	}
}
