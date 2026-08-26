package models

import "time"

// Skill 表示一个已安装的共享 Skill（设计文档 12.1）
type Skill struct {
	ID              string    `json:"id"`
	DisplayName     string    `json:"displayName"`
	SourceURL       string    `json:"sourceUrl"`
	RepositoryOwner string    `json:"repositoryOwner"`
	RepositoryName  string    `json:"repositoryName"`
	RepositoryPath  string    `json:"repositoryPath"`
	CurrentVersionID string   `json:"currentVersionId"`
	// Note 是用户为 Skill 设置的备注名（别名，可为空，非数据库必填）
	Note             string   `json:"note"`
	// DisplayVersion 是当前版本的显示名（由 ListSkills 联表填充，非数据库列）
	DisplayVersion  string    `json:"displayVersion"`
	// InstalledAt 是当前版本的安装时间（由 ListSkills 联表填充，非数据库列）
	InstalledAt     time.Time `json:"installedAt"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// SkillVersion 表示 Skill 的一个不可变版本（设计文档 12.2）
type SkillVersion struct {
	ID            string    `json:"id"`
	SkillID       string    `json:"skillId"`
	DisplayVersion string   `json:"displayVersion"`
	GitRef        string    `json:"gitRef"`
	CommitSHA     string    `json:"commitSha"`
	Checksum      string    `json:"checksum"`
	InstallPath   string    `json:"installPath"`
	InstalledAt   time.Time `json:"installedAt"`
}

// Agent 表示一个已启用的 Agent 及其配置（设计文档 12.3）
type Agent struct {
	ID          string    `json:"id"`
	AdapterID   string    `json:"adapterId"`
	DisplayName string    `json:"displayName"`
	SkillsPath  string    `json:"skillsPath"`
	Enabled     bool      `json:"enabled"`
	Detected    bool      `json:"detected"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ManagedLink 表示管理器为某个 Agent 创建的文件链接（设计文档 12.4）
type ManagedLink struct {
	ID             string    `json:"id"`
	AgentID        string    `json:"agentId"`
	SkillID        string    `json:"skillId"`
	SkillVersionID string    `json:"skillVersionId"`
	SourcePath     string    `json:"sourcePath"`
	TargetPath     string    `json:"targetPath"`
	LinkMode       string    `json:"linkMode"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AppConfig 表示应用自身配置（共享目录位置和界面设置）
type AppConfig struct {
	SharedDir string `json:"sharedDir"`
	// DownloadMirror 是可选的 GitHub 下载加速镜像前缀（如 https://ghproxy.com/）
	DownloadMirror string `json:"downloadMirror"`
	// GitHubAuthMethod 是 GitHub 授权方式："" 未设置 | "token" 手动 token | "oauth" 登录授权
	GitHubAuthMethod string `json:"gitHubAuthMethod"`
	// GitHubToken 是手动填写的 GitHub token（用于提升 API 配额）。仅本地保存。
	GitHubToken string `json:"gitHubToken"`
	// GitHubUserLogin 是已授权账号的用户名（login），用于设置页展示。仅本地保存。
	GitHubUserLogin string `json:"gitHubUserLogin"`
	// GitHubUserName 是已授权账号的显示名（真实姓名，可空），用于设置页展示。仅本地保存。
	GitHubUserName string `json:"gitHubUserName"`
}
