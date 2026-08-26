package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"agent-skill-manager/internal/models"
)

// AppName 是应用名称，用于定位配置目录
const AppName = "AgentSkillManager"

// Manager 管理应用自身的配置（共享目录位置和界面设置）
type Manager struct {
	configDir string
	configFile string
	config    *models.AppConfig
}

// NewManager 创建配置管理器，并确定配置目录位置
func NewManager() (*Manager, error) {
	dir, err := configDirectory()
	if err != nil {
		return nil, err
	}
	m := &Manager{
		configDir:  dir,
		configFile: filepath.Join(dir, "config.json"),
		config:     &models.AppConfig{},
	}
	// 确保配置目录存在
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return m, nil
}

// configDirectory 返回应用配置目录
// Windows: %LOCALAPPDATA%\AgentSkillManager
// macOS:   ~/Library/Application Support/AgentSkillManager
func configDirectory() (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, AppName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", AppName), nil
}

// Load 从磁盘加载配置；如果不存在则返回空配置
func (m *Manager) Load() error {
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, m.config)
}

// Save 将当前配置写入磁盘
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configFile, data, 0o644)
}

// ConfigDir 返回配置目录路径
func (m *Manager) ConfigDir() string {
	return m.configDir
}

// GetConfig 返回当前配置对象
func (m *Manager) GetConfig() *models.AppConfig {
	return m.config
}

// SetSharedDir 设置共享 Skill 目录并持久化
func (m *Manager) SetSharedDir(path string) error {
	if path == "" {
		return errors.New("共享目录不能为空")
	}
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	m.config.SharedDir = path
	return m.Save()
}

// SetDownloadMirror 设置 GitHub 下载加速镜像前缀并持久化
func (m *Manager) SetDownloadMirror(mirror string) error {
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	m.config.DownloadMirror = strings.TrimSpace(mirror)
	return m.Save()
}

// SetGitHubAuthorization 设置 GitHub 授权方式并持久化。
// method 取值："token"（手填）或 "oauth"（登录授权）；token 为对应的授权 token。
// 无论哪种方式，token 都写入 GitHubToken 字段持久化，method 仅用于区分来源。
func (m *Manager) SetGitHubAuthorization(method, token string) error {
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	if method != "token" && method != "oauth" {
		return errors.New("不支持的授权方式")
	}
	m.config.GitHubAuthMethod = method
	m.config.GitHubToken = strings.TrimSpace(token)
	return m.Save()
}

// SetGitHubAccount 持久化已授权账号信息（用于设置页展示），仅本地保存。
// login 为 GitHub 用户名，name 为显示名（可空）。
func (m *Manager) SetGitHubAccount(login, name string) error {
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	m.config.GitHubUserLogin = login
	m.config.GitHubUserName = name
	return m.Save()
}

// ClearGitHubAuthorization 清除 GitHub 授权（恢复为未授权/受限状态）
func (m *Manager) ClearGitHubAuthorization() error {
	if m.config == nil {
		m.config = &models.AppConfig{}
	}
	m.config.GitHubAuthMethod = ""
	m.config.GitHubToken = ""
	m.config.GitHubUserLogin = ""
	m.config.GitHubUserName = ""
	return m.Save()
}
