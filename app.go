package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"agent-skill-manager/internal/agents"
	"agent-skill-manager/internal/config"
	"agent-skill-manager/internal/github"
	"agent-skill-manager/internal/installer"
	"agent-skill-manager/internal/models"
	"agent-skill-manager/internal/platform"
	"agent-skill-manager/internal/security"
	"agent-skill-manager/internal/skills"
	"agent-skill-manager/internal/storage"

	browser "github.com/pkg/browser"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是应用服务层，持有配置与存储等核心服务
type App struct {
	ctx         context.Context
	cfg         *config.Manager
	store       *storage.Store
	ghToken     string // 当前生效的 GitHub token（来自手填 token 或登录授权），运行时内存态
	ghUserLogin string // 当前授权账号的 GitHub 用户名（login），用于设置页展示
	ghUserName  string // 当前授权账号的显示名（真实姓名，可空），用于设置页展示
	oauthCancel context.CancelFunc
}

// NewApp 创建应用服务实例
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时初始化核心服务
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 初始化配置管理器
	cfg, err := config.NewManager()
	if err != nil {
		println("初始化配置失败:", err.Error())
		return
	}
	if err := cfg.Load(); err != nil {
		println("加载配置失败:", err.Error())
		return
	}
	a.cfg = cfg

	// 加载 GitHub token（来自手填 token 或登录授权）
	if cfg.GetConfig().GitHubAuthMethod == "token" {
		a.ghToken = cfg.GetConfig().GitHubToken
	} else if cfg.GetConfig().GitHubAuthMethod == "oauth" {
		a.ghToken = cfg.GetConfig().GitHubToken
	}
	// 加载已授权账号信息（用于设置页展示）
	a.ghUserLogin = cfg.GetConfig().GitHubUserLogin
	a.ghUserName = cfg.GetConfig().GitHubUserName
	// 兼容历史已授权但未缓存账号信息的情况：尝试补齐一次
	if a.ghToken != "" && a.ghUserLogin == "" {
		a.fetchAccountAndCache()
	}

	// 若已配置共享目录，则打开数据库
	sharedDir := cfg.GetConfig().SharedDir
	if sharedDir != "" {
		if err := a.openStore(sharedDir); err != nil {
			println("初始化数据库失败:", err.Error())
		}
	}
}

// openStore 打开共享目录对应的数据库
func (a *App) openStore(sharedDir string) error {
	store, err := storage.Open(sharedDir)
	if err != nil {
		return err
	}
	a.store = store
	return nil
}

// ---------- 前端绑定方法 ----------

// GetAppInfo 返回应用基础信息（供前端展示和判断）
func (a *App) GetAppInfo() map[string]string {
	configDir := ""
	sharedDir := ""
	downloadMirror := ""
	ghAuthMethod := ""
	ghTokenSet := ""
	if a.cfg != nil {
		configDir = a.cfg.ConfigDir()
		sharedDir = a.cfg.GetConfig().SharedDir
		downloadMirror = a.cfg.GetConfig().DownloadMirror
		ghAuthMethod = a.cfg.GetConfig().GitHubAuthMethod
		if ghAuthMethod == "token" {
			ghTokenSet = boolString(a.cfg.GetConfig().GitHubToken != "")
		}
		if ghAuthMethod == "oauth" {
			// oauth 方式下是否有 token 由 a.ghToken 决定
			ghTokenSet = boolString(a.ghToken != "")
		}
	}
	return map[string]string{
		"name":                "Agent Skill Manager",
		"version":             "0.1.0",
		"goos":                runtime.GOOS,
		"configDir":           configDir,
		"sharedDir":           sharedDir,
		"downloadMirror":      downloadMirror,
		"isConfigured":        boolString(sharedDir != ""),
		"gitHubAuthMethod":    ghAuthMethod,
		"gitHubTokenSet":      ghTokenSet,
		"gitHubUserLogin":     a.ghUserLogin,
		"gitHubUserName":      a.ghUserName,
	}
}

// SetDownloadMirror 设置 GitHub 下载加速镜像前缀并持久化
func (a *App) SetDownloadMirror(mirror string) error {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}
	if err := a.cfg.SetDownloadMirror(mirror); err != nil {
		return &AppError{Message: "保存镜像配置失败: " + err.Error()}
	}
	return nil
}

// SetGitHubToken 手动填写 GitHub token 并保存。
func (a *App) SetGitHubToken(token string) error {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}
	clean := strings.TrimSpace(token)
	if clean == "" {
		return &AppError{Message: "token 不能为空"}
	}
	if err := a.cfg.SetGitHubAuthorization("token", clean); err != nil {
		return &AppError{Message: "保存 token 失败: " + err.Error()}
	}
	a.ghToken = clean
	// 拉取并缓存授权账号信息，用于设置页展示
	a.fetchAccountAndCache()
	return nil
}

// StartGitHubOAuth 发起 GitHub 登录授权（设备流），返回用户需要在浏览器
// 打开地址 + 输入的验证码。调用方需轮询 CheckGitHubOAuth 获取授权结果。
func (a *App) StartGitHubOAuth() (map[string]string, error) {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return nil, &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}
	ctx, cancel := context.WithCancel(a.ctx)
	// 保存 cancel，供轮询超时取消用
	a.oauthCancel = cancel

	dev, err := github.StartDeviceAuth(ctx)
	if err != nil {
		return nil, &AppError{Message: "发起 GitHub 授权失败: " + err.Error()}
	}
	// 打开浏览器让用户完成授权（使用 OS 原生方式，更可靠）
	_ = browser.OpenURL(dev.VerifyURI)
	return map[string]string{
		"verificationUri": dev.VerifyURI,
		"userCode":        dev.UserCode,
		"deviceCode":      dev.DeviceCode,
	}, nil
}

// CheckGitHubOAuth 轮询一次登录授权结果。
// 若用户已完成授权，则保存 token 并返回 completed=true。
func (a *App) CheckGitHubOAuth(deviceCode string) (map[string]string, error) {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return nil, &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
	defer cancel()

	token, err := github.PollDeviceAuthPoll(ctx, deviceCode, 5)
	if err != nil {
		return nil, &AppError{Message: "等待 GitHub 授权超时或失败: " + err.Error()}
	}
	if err := a.cfg.SetGitHubAuthorization("oauth", token); err != nil {
		return nil, &AppError{Message: "保存授权失败: " + err.Error()}
	}
	if a.oauthCancel != nil {
		a.oauthCancel()
		a.oauthCancel = nil
	}
	a.ghToken = token
	// 拉取并缓存授权账号信息，用于设置页展示
	a.fetchAccountAndCache()
	return map[string]string{"completed": boolString(true)}, nil
}

// ClearGitHubAuth 清除当前 GitHub 授权信息。
func (a *App) ClearGitHubAuth() error {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}
	if err := a.cfg.ClearGitHubAuthorization(); err != nil {
		return &AppError{Message: "清除授权失败: " + err.Error()}
	}
	a.ghToken = ""
	a.ghUserLogin = ""
	a.ghUserName = ""
	return nil
}

// SetSharedDir 设置共享 Skill 目录。首次调用时还会初始化数据库。
func (a *App) SetSharedDir(path string) error {
	if a.cfg == nil {
		cfg, err := config.NewManager()
		if err != nil {
			return &AppError{Message: "初始化配置失败: " + err.Error()}
		}
		a.cfg = cfg
	}

	clean := strings.TrimSpace(path)
	if clean == "" {
		return &AppError{Message: "共享目录不能为空"}
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return &AppError{Message: "无法解析目录路径: " + err.Error()}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return &AppError{Message: "无法创建目录: " + err.Error()}
	}

	if err := a.cfg.SetSharedDir(abs); err != nil {
		return &AppError{Message: "保存配置失败: " + err.Error()}
	}
	if a.store == nil {
		if err := a.openStore(abs); err != nil {
			return &AppError{Message: "初始化数据库失败: " + err.Error()}
		}
	}
	return nil
}

// ListSkills 返回所有已安装的共享 Skill
func (a *App) ListSkills() ([]models.Skill, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	return a.store.ListSkills(context.Background())
}

// ListAgents 返回所有已登记的 Agent
func (a *App) ListAgents() ([]models.Agent, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	return a.store.ListAgents(context.Background())
}

// DetectAgents 检测本机已安装的 Agent（如 Codex、OpenCode）并登记入库。
// 返回更新后的 Agent 列表。
func (a *App) DetectAgents() ([]models.Agent, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	ctx := context.Background()

	// 读取现有登记，保留用户已启用的状态
	existing, _ := a.store.ListAgents(ctx)
	enabledByID := make(map[string]bool, len(existing))
	for _, ag := range existing {
		enabledByID[ag.ID] = ag.Enabled
	}

	reg := agents.DefaultRegistry()
	detected := reg.DetectAll(ctx)
	for id, inst := range detected {
		adapter, ok := reg.Get(id)
		if !ok {
			continue
		}
		ag := &models.Agent{
			ID:          id,
			AdapterID:   id,
			DisplayName: adapter.DisplayName(),
			SkillsPath:  inst.SkillsPath,
			Enabled:     enabledByID[id], // 保留原有启用状态，新检测到的默认未启用
			Detected:    true,
		}
		if err := a.store.UpsertAgent(ctx, ag); err != nil {
			return nil, err
		}
	}

	return a.store.ListAgents(ctx)
}

// SetAgentEnabled 启用或停用某个 Agent，启用后该 Agent 会自动使用全部共享 Skill。
// 返回更新后的 Agent 列表。
func (a *App) SetAgentEnabled(agentID string, enabled bool) ([]models.Agent, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	ctx := context.Background()

	agentsList, err := a.store.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, ag := range agentsList {
		if ag.ID == agentID {
			ag.Enabled = enabled
			if err := a.store.UpsertAgent(ctx, &ag); err != nil {
				return nil, err
			}
			found = true
			// 启用：立即把共享库中所有已装技能以链接方式同步进来（不动工具自带的技能）
			// 关闭：移除之前同步到该工具的所有共享技能快捷方式（同样不动工具自带技能）
			if enabled {
				if failMsg := a.syncAllSkillsToAgent(ctx, &ag); failMsg != "" {
					// 工具已启用，技能可能部分未同步；返回错误提示，便于用户知晓
					agentsList, err = a.store.ListAgents(ctx)
					if err != nil {
						return nil, err
					}
					return agentsList, &AppError{Message: failMsg}
				}
			} else {
				if failMsg := a.removeAllSharedFromAgent(ctx, &ag); failMsg != "" {
					agentsList, err = a.store.ListAgents(ctx)
					if err != nil {
						return nil, err
					}
					return agentsList, &AppError{Message: failMsg}
				}
			}
			break
		}
	}
	if !found {
		return nil, &AppError{Message: "未找到指定的 Agent: " + agentID}
	}
	return a.store.ListAgents(ctx)
}

// syncAllSkillsToAgent 将共享库中所有已装技能，以链接方式同步到指定工具。
// 同名冲突（工具自带技能）会跳过并保留原样；建不了链接的技能会记录为失败。
// 返回失败提示；全部成功或全部被冲突跳过时返回空字符串。
func (a *App) syncAllSkillsToAgent(ctx context.Context, agent *models.Agent) string {
	skills, err := a.store.ListSkills(ctx)
	if err != nil {
		return "查询共享技能失败: " + err.Error()
	}
	linker := platform.NewLinker()
	var failed []string
	for i := range skills {
		sk := &skills[i]
		// 链接名与卸载时一致：优先用显示名，其次用 ID 末段
		name := sk.DisplayName
		if name == "" {
			name = skillNameFromID(sk.ID)
		}
		versionDir := a.skillVersionDir(ctx, sk)
		if versionDir == "" {
			continue
		}
		target := filepath.Join(agent.SkillsPath, name)
		if _, err := linker.Create(versionDir, target); err != nil {
			// 同名冲突：目标已存在（工具自带或之前已建），跳过并保留原样，不算失败
			var conflict *platform.ConflictError
			if errors.As(err, &conflict) {
				continue
			}
			failed = append(failed, name+": "+err.Error())
		}
	}
	if len(failed) > 0 {
		return "已启用，但部分共享技能同步失败（未使用复制）：" + strings.Join(failed, "；")
	}
	return ""
}

// skillVersionDir 返回某个技能的当前版本安装目录（优先取数据库记录的真实安装路径）
func (a *App) skillVersionDir(ctx context.Context, sk *models.Skill) string {
	if sk.CurrentVersionID != "" {
		if v, err := a.store.GetVersion(ctx, sk.CurrentVersionID); err == nil && v != nil && v.InstallPath != "" {
			return v.InstallPath
		}
	}
	return a.installDirForSkill(sk)
}

// removeAllSharedFromAgent 移除指定工具目录中由本程序同步过去的全部共享技能快捷方式。
// 只删除程序创建的链接，绝不动工具自带或其他内容。返回失败提示；全部成功或无事可做时返回空字符串。
func (a *App) removeAllSharedFromAgent(ctx context.Context, agent *models.Agent) string {
	skills, err := a.store.ListSkills(ctx)
	if err != nil {
		return "查询共享技能失败: " + err.Error()
	}
	linker := platform.NewLinker()
	var failed []string
	for i := range skills {
		sk := &skills[i]
		name := sk.DisplayName
		if name == "" {
			name = skillNameFromID(sk.ID)
		}
		target := filepath.Join(agent.SkillsPath, name)

		// 只移除程序创建的链接；非链接或不存在则跳过
		info, err := linker.Inspect(target)
		if err != nil || !info.Exists {
			continue
		}
		// RemoveManaged 会校验指向（指向共享库才删），并删除链接本身（不删来源）
		if err := linker.RemoveManaged(target, a.skillVersionDir(ctx, sk)); err != nil {
			failed = append(failed, name+": "+err.Error())
			continue
		}
		// 移除后若残留空目录则一并清理
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			// 非空目录不强制删除
		}
	}
	if len(failed) > 0 {
		return "部分共享技能移除失败：" + strings.Join(failed, "；")
	}
	return ""
}

// SetAgentSkillsPath 手动修改某个 Agent 的技能目录（用于工具目录被改动或自定义的情况）。
// 返回更新后的 Agent 列表。
func (a *App) SetAgentSkillsPath(agentID, skillsPath string) ([]models.Agent, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	skillsPath = strings.TrimSpace(skillsPath)
	if skillsPath == "" {
		return nil, &AppError{Message: "技能目录不能为空"}
	}
	// 尝试创建目录，确保路径有效可用
	if err := os.MkdirAll(skillsPath, 0o755); err != nil {
		return nil, &AppError{Message: "无法使用该目录: " + err.Error()}
	}
	ctx := context.Background()
	agentsList, err := a.store.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, ag := range agentsList {
		if ag.ID == agentID {
			ag.SkillsPath = filepath.Clean(skillsPath)
			ag.Detected = false // 手动指定后，标记为非自动检测路径
			if err := a.store.UpsertAgent(ctx, &ag); err != nil {
				return nil, err
			}
			found = true
			break
		}
	}
	if !found {
		return nil, &AppError{Message: "未找到指定的 Agent: " + agentID}
	}
	return a.store.ListAgents(ctx)
}

// PickAgentSkillsDir 弹出系统对话框选择某个工具的 Skill 目录，返回选中路径。
func (a *App) PickAgentSkillsDir() (string, error) {
	if a.ctx == nil {
		return "", &AppError{Message: "应用尚未就绪"}
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择该工具的 Skill 目录",
	})
	if err != nil {
		return "", &AppError{Message: "选择文件夹失败: " + err.Error()}
	}
	return dir, nil
}

// ---------- 安装相关绑定方法 ----------

// ScanResult 是扫描仓库后的结果
type ScanResult struct {
	Owner       string            `json:"owner"`
	Repo        string            `json:"repo"`
	SubPath     string            `json:"subPath"`
	DefaultRef  string            `json:"defaultRef"`
	Description string            `json:"description"`
	Stars       int               `json:"stars"`
	Versions    []github.Version  `json:"versions"`
}

// ScanRepository 解析 GitHub URL 并查询仓库信息与可用版本
func (a *App) ScanRepository(raw string) (*ScanResult, error) {
	ref, err := github.ParseURL(raw)
	if err != nil {
		return nil, &AppError{Message: err.Error()}
	}

	gh := a.newGHClient()
	ctx := context.Background()

	repoInfo, err := gh.GetRepoInfo(ctx, ref)
	if err != nil {
		return nil, &AppError{Message: "无法查询仓库: " + err.Error()}
	}

	defaultRef, err := gh.ResolveRef(ctx, ref)
	if err != nil {
		defaultRef = "main"
	}

	versions, err := gh.ListVersions(ctx, ref, 20)
	if err != nil || versions == nil {
		versions = []github.Version{}
	}

	return &ScanResult{
		Owner:       ref.Owner,
		Repo:        ref.Repo,
		SubPath:     ref.SubPath,
		DefaultRef:  defaultRef,
		Description: repoInfo.Description,
		Stars:       repoInfo.Stars,
		Versions:    versions,
	}, nil
}

// RepoSkill 是仓库内扫描出的一个技能
type RepoSkill struct {
	Name    string `json:"name"`    // 技能目录名（Agent 目录中显示用）
	RelPath string `json:"relPath"` // 技能在仓库内的相对路径（安装时定位用）
}

// ListRepositorySkills 下载并扫描 GitHub 仓库，列出其中所有可安装的技能。
// 返回空列表表示仓库中无 SKILL.md（如 awesome 清单类仓库）。
func (a *App) ListRepositorySkills(raw string) ([]RepoSkill, error) {
	ref, err := github.ParseURL(raw)
	if err != nil {
		return nil, &AppError{Message: err.Error()}
	}

	gh := a.newGHClient()
	ctx := context.Background()

	// 确定分支
	gitRef := ref.Ref
	if gitRef == "" {
		gitRef, err = gh.ResolveRef(ctx, ref)
		if err != nil {
			gitRef = "main"
		}
	}

	// 下载并解压到临时目录
	archiveURL, err := gh.ArchiveURL(ctx, ref, gitRef)
	if err != nil {
		return nil, &AppError{Message: "获取下载地址失败: " + err.Error()}
	}
	tmpDir, err := os.MkdirTemp("", "skill-preview-*")
	if err != nil {
		return nil, &AppError{Message: "创建临时目录失败: " + err.Error()}
	}
	defer os.RemoveAll(tmpDir)

	if err := security.DownloadAndExtract(archiveURL, tmpDir); err != nil {
		return nil, &AppError{Message: "下载仓库失败: " + err.Error()}
	}

	root, err := security.FindRootDir(tmpDir)
	if err != nil {
		return nil, &AppError{Message: "解析仓库内容失败: " + err.Error()}
	}

	// 扫描所有 SKILL.md，每个父目录为一个技能
	scanner := skills.NewScanner()
	files, err := scanner.Scan(root)
	if err != nil {
		return nil, &AppError{Message: "扫描技能失败: " + err.Error()}
	}

	seen := map[string]bool{}
	result := make([]RepoSkill, 0, len(files))
	for _, f := range files {
		name := filepath.Base(f.Dir)
		if name == "" || seen[name] {
			continue
		}
		rel, _ := filepath.Rel(root, f.Dir)
		seen[name] = true
		result = append(result, RepoSkill{
			Name:    name,
			RelPath: filepath.ToSlash(rel),
		})
	}
	if result == nil {
		result = []RepoSkill{}
	}
	return result, nil
}

// InstallRequest 是安装请求
type InstallRequest struct {
	URL    string `json:"url"`
	GitRef string `json:"gitRef"` // 可空，默认用默认分支
	Note   string `json:"note"`   // 可空，用户为 Skill 设置的备注名（别名）
	// SkillPath 是仓库内要安装的单个技能的相对路径（由技能预览列表提供）。
	// 为空时回退到旧行为：安装仓库扫描到的第一个技能。
	SkillPath string `json:"skillPath"`
}

// InstallResult 是安装结果
type InstallResult struct {
	SkillID      string `json:"skillId"`
	InstallPath  string `json:"installPath"`
	SyncedAgents []string `json:"syncedAgents"`
	Conflicts    []string `json:"conflicts"`
}

// InstallSkill 执行完整安装流程并同步到所有已启用 Agent
func (a *App) InstallSkill(req InstallRequest) (*InstallResult, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}

	ref, err := github.ParseURL(req.URL)
	if err != nil {
		return nil, &AppError{Message: err.Error()}
	}

	gh := a.newGHClient()
	ctx := context.Background()

	// 确定 gitRef 和 commit SHA
	gitRef := req.GitRef
	if gitRef == "" {
		gitRef, err = gh.ResolveRef(ctx, ref)
		if err != nil {
			return nil, &AppError{Message: "无法确定分支: " + err.Error()}
		}
	}
	// 获取精确 commit SHA
	commitSHA, err := gh.ResolveCommitSHA(ctx, ref, gitRef)
	if err != nil {
		commitSHA = gitRef
	}

	// 执行安装
	sharedDir := a.cfg.GetConfig().SharedDir
	inst := installer.New(sharedDir)
	result, err := inst.Install(ctx, installer.InstallOptions{
		RepoRef:      ref,
		GitRef:       gitRef,
		Commit:       commitSHA,
		SkillRelPath: req.SkillPath,
	})
	if err != nil {
		return nil, &AppError{Message: "安装失败: " + err.Error()}
	}

	// 记录到数据库（含版本记录）
	if err := a.recordInstall(ctx, ref, result, gitRef, commitSHA, req.Note); err != nil {
		return nil, &AppError{Message: "记录安装状态失败: " + err.Error()}
	}

	// 同步到所有已启用 Agent
	synced, conflicts, err := a.syncToAgents(ctx, result.InstallPath, result.SkillID)
	if err != nil {
		return nil, &AppError{Message: "同步到 Agent 失败: " + err.Error()}
	}

	return &InstallResult{
		SkillID:      result.SkillID,
		InstallPath:  result.InstallPath,
		SyncedAgents: synced,
		Conflicts:    conflicts,
	}, nil
}

// LocalInstallResult 是本地安装返回给前端的结构
type LocalInstallResult struct {
	SkillID      string   `json:"skillId"`
	SkillName    string   `json:"skillName"`
	InstallPath  string   `json:"installPath"`
	SyncedAgents []string `json:"syncedAgents"`
	Conflicts    []string `json:"conflicts"`
}

// InstallLocalSkill 从本地文件夹或 SKILL.md 文件安装 Skill，并同步到已启用 Agent。
// 采用覆盖更新策略：同名 Skill 会先移除旧链接再重新创建。
func (a *App) InstallLocalSkill(path string) (*LocalInstallResult, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}

	sharedDir := a.cfg.GetConfig().SharedDir
	inst := installer.New(sharedDir)
	ctx := context.Background()

	results, err := inst.InstallLocal(ctx, path)
	if err != nil {
		return nil, &AppError{Message: "本地安装失败: " + err.Error()}
	}
	if len(results) == 0 {
		return nil, &AppError{Message: "未找到可安装的 Skill"}
	}

	// 逐个记录到数据库（含覆盖更新），再同步到 Agent
	var synced []string
	var conflicts []string
	for i := range results {
		local := &results[i]
		if err := a.recordLocalInstall(ctx, local); err != nil {
			return nil, &AppError{Message: "记录本地安装状态失败: " + err.Error()}
		}
		// 每个 Skill 同步到已启用 Agent
		s, c, err := a.syncLocalToAgents(ctx, local)
		if err != nil {
			return nil, &AppError{Message: "同步到 Agent 失败: " + err.Error()}
		}
		synced = append(synced, s...)
		conflicts = append(conflicts, c...)
	}

	first := &results[0]
	return &LocalInstallResult{
		SkillID:      first.SkillID,
		SkillName:    first.SkillName,
		InstallPath:  first.InstallPath,
		SyncedAgents: synced,
		Conflicts:    conflicts,
	}, nil
}

// UninstallResult 是卸载结果
type UninstallResult struct {
	SkillID string   `json:"skillId"`
	Removed []string `json:"removed"`  // 从哪些 Agent 移除了链接
	Failed  []string `json:"failed"`   // 哪些 Agent 移除失败
}

// UninstallSkill 卸载一个 Skill：
// 1. 从所有已启用 Agent 移除指向该 Skill 的链接
// 2. 删除共享目录中的安装目录
// 3. 从数据库删除记录
func (a *App) UninstallSkill(skillID string) (*UninstallResult, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	ctx := context.Background()

	// 查询 Skill 记录
	skill, err := a.store.GetSkill(ctx, skillID)
	if err != nil {
		return nil, &AppError{Message: "查询 Skill 失败: " + err.Error()}
	}
	if skill == nil {
		return nil, &AppError{Message: "未找到该 Skill: " + skillID}
	}

	// 1. 从已启用 Agent 移除链接
	removed, failed := a.removeFromAgents(ctx, skill)

	// 2. 删除共享目录中的安装目录
	installDir := a.installDirForSkill(skill)
	if installDir != "" {
		if err := os.RemoveAll(installDir); err != nil {
			// 目录删除失败不阻断，记录为 failed
			failed = append(failed, "删除安装目录失败: "+err.Error())
		}
	}

	// 3. 从数据库删除
	if err := a.store.DeleteSkill(ctx, skillID); err != nil {
		return nil, &AppError{Message: "删除数据库记录失败: " + err.Error()}
	}

	return &UninstallResult{
		SkillID: skillID,
		Removed: removed,
		Failed:  failed,
	}, nil
}

// removeFromAgents 从所有已启用 Agent 的 skills 目录移除指向该 Skill 的链接
func (a *App) removeFromAgents(ctx context.Context, skill *models.Skill) (removed, failed []string) {
	agentList, err := a.store.ListAgents(ctx)
	if err != nil {
		failed = append(failed, "查询 Agent 失败: "+err.Error())
		return removed, failed
	}

	// Skill 在 Agent 目录中的链接名 = 显示名
	linkName := skill.DisplayName
	if linkName == "" {
		linkName = skillNameFromID(skill.ID)
	}

	linker := platform.NewLinker()

	for _, ag := range agentList {
		if !ag.Enabled {
			continue
		}
		target := filepath.Join(ag.SkillsPath, linkName)

		// 检查目标是否存在且是链接
		info, err := linker.Inspect(target)
		if err != nil || !info.Exists {
			continue
		}

		// 移除链接。卸载技能本就要清掉它的所有痕迹：
		// 1) 先尝试 RemoveManaged（校验指向后删除，兼顾安全）；
		// 2) 若因指向校验失败（例如技能更新过、指向旧版本目录）删不掉，且目标是链接，
		//    则直接删除链接本身——链接删除不触碰来源、绝不影响用户内容。
		if err := linker.RemoveManaged(target, a.skillVersionDir(ctx, skill)); err != nil {
			// 仅当确认目标是链接时才直接删除；普通目录（可能是用户内容）不强行删
			if info.Exists && (info.Mode == platform.LinkModeSymlink || info.Mode == platform.LinkModeJunction) {
				if rmErr := os.RemoveAll(target); rmErr != nil {
					failed = append(failed, ag.DisplayName+": "+err.Error())
					continue
				}
			} else {
				failed = append(failed, ag.DisplayName+": "+err.Error())
				continue
			}
		}
		// 若是托管复制，移除后目标目录可能还在，再尝试删除空目录
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			// 非空目录不强制删除
		}
		removed = append(removed, ag.DisplayName)
	}

	if removed == nil {
		removed = []string{}
	}
	if failed == nil {
		failed = []string{}
	}
	return removed, failed
}

// installDirForSkill 返回共享目录中该 Skill 的安装目录
// 本地安装：sharedDir/local/<name>
// GitHub 安装：sharedDir/packages/<id路径>/<commit>（无 commit 时返回 packages 根路径下的 id 目录）
func (a *App) installDirForSkill(skill *models.Skill) string {
	sharedDir := a.cfg.GetConfig().SharedDir
	if sharedDir == "" {
		return ""
	}

	// 本地安装：SourceURL 就是安装路径
	if skill.RepositoryOwner == "local" {
		return filepath.Join(sharedDir, "local", skill.RepositoryName)
	}

	// GitHub 安装：packages/<id> 目录
	pkgRoot := filepath.Join(sharedDir, "packages", filepath.FromSlash(skill.ID))
	return pkgRoot
}

// PickLocalDirectory 弹出系统对话框选择本地文件夹，返回选中路径。
// CheckUpdateResult 是检查更新的结果
type CheckUpdateResult struct {
	SkillID         string `json:"skillId"`
	IsLocal         bool   `json:"isLocal"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	HasUpdate       bool   `json:"hasUpdate"`
	CurrentCommit   string `json:"currentCommit"`
	LatestCommit    string `json:"latestCommit"`
	UpdateNotes     string `json:"updateNotes"` // 远程最新提交的更新说明
	CheckError      string `json:"checkError"` // 检查失败时的错误信息（如网络问题）
}

// CheckUpdate 检查某个 Skill 是否有更新。
// 仅对 GitHub 来源有效；本地 Skill 返回无更新。
func (a *App) CheckUpdate(skillID string) (*CheckUpdateResult, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	ctx := context.Background()

	skill, err := a.store.GetSkill(ctx, skillID)
	if err != nil {
		return nil, &AppError{Message: "查询 Skill 失败: " + err.Error()}
	}
	if skill == nil {
		return nil, &AppError{Message: "未找到该 Skill: " + skillID}
	}

	result := &CheckUpdateResult{
		SkillID:        skill.ID,
		IsLocal:        skill.RepositoryOwner == "local",
		CurrentVersion: "",
	}

	// 获取当前版本
	if skill.CurrentVersionID != "" {
		if cur, err := a.store.GetVersion(ctx, skill.CurrentVersionID); err == nil && cur != nil {
			result.CurrentVersion = cur.DisplayVersion
			result.CurrentCommit = cur.CommitSHA
		}
	}

	// 本地 Skill 无远程源，无法检查
	if result.IsLocal {
		result.HasUpdate = false
		result.LatestVersion = result.CurrentVersion
		return result, nil
	}

	// GitHub：查询远程最新 commit
	ref := &github.RepoRef{
		Owner:      skill.RepositoryOwner,
		Repo:       skill.RepositoryName,
		SubPath:    skill.RepositoryPath,
		Original:   skill.SourceURL,
	}
	gh := a.newGHClient()

	// 判断当前版本是 tag 还是分支（跟随版本类型），与 UpdateSkill 保持一致
	currentGitRef := ""
	if skill.CurrentVersionID != "" {
		if cur, err := a.store.GetVersion(ctx, skill.CurrentVersionID); err == nil && cur != nil {
			currentGitRef = cur.GitRef
		}
	}
	isTagMode := currentGitRef != "" && !looksLikeBranch(currentGitRef)

	// 确定要检查的目标 ref（tag 模式跟随最新 tag；分支模式跟随默认分支）
	var targetRef string
	if isTagMode {
		latestTag, err := gh.LatestTag(ctx, ref)
		if err != nil || latestTag == "" {
			result.CheckError = "无法获取仓库最新版本标签: " + err.Error()
			return result, nil
		}
		targetRef = latestTag
	} else {
		targetRef, err = gh.ResolveRef(ctx, ref)
		if err != nil {
			result.CheckError = "无法确定远程分支: " + err.Error()
			return result, nil
		}
	}

	commitInfo, err := gh.GetCommitInfo(ctx, ref, targetRef)
	if err != nil {
		result.CheckError = "无法获取远程最新版本: " + err.Error()
		return result, nil
	}

	result.LatestCommit = commitInfo.SHA
	result.LatestVersion = shortSHA(commitInfo.SHA)
	result.UpdateNotes = strings.TrimSpace(commitInfo.Message)

	// 对比当前 commit 与最新 commit
	if result.CurrentCommit != "" && result.CurrentCommit != commitInfo.SHA {
		result.HasUpdate = true
	}

	return result, nil
}

// shortSHA 返回 commit SHA 前 7 位
func shortSHA(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}

// UpdateSkill 更新一个 GitHub Skill 到远程最新版本。
// 流程：获取最新 commit -> 下载新版本 -> 写新版本记录 -> 切换链接到新版本 -> 重新同步到 Agent。
func (a *App) UpdateSkill(skillID string) (*InstallResult, error) {
	if a.store == nil {
		return nil, &AppError{Message: "尚未配置共享目录"}
	}
	ctx := context.Background()

	skill, err := a.store.GetSkill(ctx, skillID)
	if err != nil {
		return nil, &AppError{Message: "查询 Skill 失败: " + err.Error()}
	}
	if skill == nil {
		return nil, &AppError{Message: "未找到该 Skill: " + skillID}
	}
	if skill.RepositoryOwner == "local" {
		return nil, &AppError{Message: "本地 Skill 不支持在线更新，请通过本地安装覆盖更新"}
	}

	// 构造 RepoRef
	ref := &github.RepoRef{
		Owner:    skill.RepositoryOwner,
		Repo:     skill.RepositoryName,
		SubPath:  skill.RepositoryPath,
		Original: skill.SourceURL,
	}

	gh := a.newGHClient()

	// 判断当前版本是 tag 还是分支（跟随版本类型）
	currentGitRef := ""
	if skill.CurrentVersionID != "" {
		if cur, err := a.store.GetVersion(ctx, skill.CurrentVersionID); err == nil && cur != nil {
			currentGitRef = cur.GitRef
		}
	}
	isTagMode := currentGitRef != "" && !looksLikeBranch(currentGitRef)

	// 确定要更新到的目标 ref
	var gitRef string
	if isTagMode {
		// 跟随 tag：更新到最新 tag
		latestTag, err := gh.LatestTag(ctx, ref)
		if err != nil || latestTag == "" {
			return nil, &AppError{Message: "无法获取仓库最新版本标签: " + err.Error()}
		}
		gitRef = latestTag
	} else {
		// 跟随分支：更新到默认分支最新 commit
		gitRef, err = gh.ResolveRef(ctx, ref)
		if err != nil {
			return nil, &AppError{Message: "无法确定远程分支: " + err.Error()}
		}
	}

	latestCommit, err := gh.ResolveCommitSHA(ctx, ref, gitRef)
	if err != nil {
		return nil, &AppError{Message: "无法获取远程最新版本: " + err.Error()}
	}

	// 检查是否真的需要更新（当前版本 ID 已是目标 commit 则无需更新）
	if skill.CurrentVersionID == latestCommit {
		return nil, &AppError{Message: "已是最新版本，无需更新"}
	}

	// 下载新版本到新版本目录
	sharedDir := a.cfg.GetConfig().SharedDir
	inst := installer.New(sharedDir)
	result, err := inst.Install(ctx, installer.InstallOptions{
		RepoRef: ref,
		GitRef:  gitRef,
		Commit:  latestCommit,
	})
	if err != nil {
		return nil, &AppError{Message: "下载新版本失败: " + err.Error()}
	}

	// 写入新版本记录，更新 currentVersionId（更新不改变备注名，传空串保留原备注）
	if err := a.recordInstall(ctx, ref, result, gitRef, latestCommit, ""); err != nil {
		return nil, &AppError{Message: "记录版本失败: " + err.Error()}
	}

	// 重新同步到已启用 Agent（先移除旧链接，再创建指向新版本）
	synced, conflicts, err := a.reSyncToAgents(ctx, skill, result.InstallPath)
	if err != nil {
		return nil, &AppError{Message: "同步到 Agent 失败: " + err.Error()}
	}

	return &InstallResult{
		SkillID:      result.SkillID,
		InstallPath:  result.InstallPath,
		SyncedAgents: synced,
		Conflicts:    conflicts,
	}, nil
}

// reSyncToAgents 更新时重新同步：先移除该 Skill 在所有已启用 Agent 的旧链接，再链接到新版本目录
func (a *App) reSyncToAgents(ctx context.Context, skill *models.Skill, newVersionDir string) (synced, conflicts []string, err error) {
	// 先移除旧链接
	_, _ = a.removeFromAgents(ctx, skill)

	// 再同步新版本
	return a.syncToAgents(ctx, newVersionDir, skill.ID)
}

// PickLocalDirectory 弹出系统对话框选择本地文件夹，返回选中路径。
func (a *App) PickLocalDirectory() (string, error) {
	if a.ctx == nil {
		return "", &AppError{Message: "应用尚未就绪"}
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择包含 SKILL.md 的本地文件夹",
	})
	if err != nil {
		return "", &AppError{Message: "选择文件夹失败: " + err.Error()}
	}
	return dir, nil
}

// PickLocalSkillFile 弹出系统对话框选择本地 SKILL.md 文件，返回选中路径。
func (a *App) PickLocalSkillFile() (string, error) {
	if a.ctx == nil {
		return "", &AppError{Message: "应用尚未就绪"}
	}
	file, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择 SKILL.md 文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Skill 文件 (*.md)", Pattern: "*.md"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", &AppError{Message: "选择文件失败: " + err.Error()}
	}
	return file, nil
}

// syncLocalToAgents 将本地安装的 Skill 同步到已启用 Agent（覆盖更新）。
// 若 Agent 目录已有同名链接，先移除旧链接再创建新链接，实现覆盖。
func (a *App) syncLocalToAgents(ctx context.Context, local *installer.LocalInstallResult) (synced, conflicts []string, err error) {
	agentList, err := a.store.ListAgents(ctx)
	if err != nil {
		return nil, nil, err
	}

	enabled := make(map[string]string)
	for _, ag := range agentList {
		if ag.Enabled {
			enabled[ag.AdapterID] = ag.SkillsPath
		}
	}
	if len(enabled) == 0 {
		return []string{}, []string{}, nil
	}

	linker := platform.NewLinker()

	for agentID, skillsDir := range enabled {
		target := filepath.Join(skillsDir, local.SkillName)

		// 覆盖更新：若已有链接，先移除（RemoveManaged 校验指向，不删来源）
		_ = linker.RemoveManaged(target, "")

		// 若 target 仍是存在的目录/文件，尝试删除；目录非空时视为用户内容冲突
		if _, err := os.Lstat(target); err == nil {
			if err := os.Remove(target); err != nil {
				// 目录非空无法删除，视为冲突，不覆盖用户内容
				conflicts = append(conflicts, agentID+"/"+local.SkillName)
				continue
			}
		}

		_, err := linker.Create(local.InstallPath, target)
		if err != nil {
			var conflict *platform.ConflictError
			if errors.As(err, &conflict) {
				conflicts = append(conflicts, agentID+"/"+local.SkillName)
				continue
			}
			conflicts = append(conflicts, agentID+"/"+local.SkillName)
			continue
		}
		synced = append(synced, agentID)
	}

	// 确保返回空数组而非 nil（避免前端 .length 崩溃）
	if synced == nil {
		synced = []string{}
	}
	if conflicts == nil {
		conflicts = []string{}
	}
	return synced, conflicts, nil
}

// UpdateSkillNote 更新某个 Skill 的备注名（别名）。备注为空时不清空已有备注。
func (a *App) UpdateSkillNote(skillID, note string) error {
	if a.store == nil {
		return &AppError{Message: "尚未配置共享目录"}
	}
	if err := a.store.UpdateSkillNote(context.Background(), skillID, note); err != nil {
		return &AppError{Message: "更新备注失败: " + err.Error()}
	}
	return nil
}

// recordInstall 将安装的 Skill 和版本记录到数据库
func (a *App) recordInstall(ctx context.Context, ref *github.RepoRef, result *installer.InstallResult, gitRef, commitSHA, note string) error {
	// 版本 ID 用 commit SHA（不可变、唯一）
	versionID := commitSHA
	if versionID == "" {
		versionID = "latest"
	}

	// 显示版本：优先用 gitRef（若是 tag/版本号），否则用 commit SHA 前 7 位
	displayVersion := displayVersionFromRef(gitRef, commitSHA)

	skill := &models.Skill{
		ID:               result.SkillID,
		DisplayName:      result.SkillName,
		SourceURL:        ref.RepoURL(),
		RepositoryOwner:  ref.Owner,
		RepositoryName:   ref.Repo,
		RepositoryPath:   ref.SubPath,
		CurrentVersionID: versionID,
		Note:             note,
	}
	if err := a.store.UpsertSkill(ctx, skill); err != nil {
		return err
	}

	// 写入版本记录（若该 commit 版本已存在则复用，避免主键冲突）
	existing, err := a.store.GetVersion(ctx, versionID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // 版本记录已存在（历史安装过该 commit），直接复用
	}

	version := &models.SkillVersion{
		ID:             versionID,
		SkillID:        result.SkillID,
		DisplayVersion: displayVersion,
		GitRef:         gitRef,
		CommitSHA:      commitSHA,
		Checksum:       "",
		InstallPath:    result.InstallPath,
	}
	return a.store.InsertVersion(ctx, version)
}

// displayVersionFromRef 根据 gitRef 和 commit 生成一个适合展示的版本号
func displayVersionFromRef(gitRef, commitSHA string) string {
	if gitRef != "" && !looksLikeBranch(gitRef) {
		return gitRef
	}
	// 使用 commit SHA 前 7 位
	if len(commitSHA) >= 7 {
		return commitSHA[:7]
	}
	if commitSHA != "" {
		return commitSHA
	}
	return gitRef
}

// looksLikeBranch 粗略判断 gitRef 是否像分支名（而非版本/tag）
// 简单规则：含 "v" 开头或纯数字版本号视为版本；否则视为分支/commit
func looksLikeBranch(ref string) bool {
	// 常见版本号模式：v1.2.3、1.2.3 等
	if len(ref) > 0 && ref[0] == 'v' {
		return false
	}
	// 全十六进制长串视为 commit
	if len(ref) >= 40 {
		return false
	}
	// 纯数字+点的版本号视为版本
	dotCount := 0
	for _, c := range ref {
		if c == '.' {
			dotCount++
		} else if c < '0' || c > '9' {
			return true // 含非数字字符，视为分支
		}
	}
	return dotCount == 0 // 无点且全数字，视为分支/其他
}

// recordLocalInstall 将本地安装的 Skill 记录到数据库（覆盖更新用 Upsert）
func (a *App) recordLocalInstall(ctx context.Context, local *installer.LocalInstallResult) error {
	// 本地安装用时间戳生成版本 ID
	versionID := "local-" + local.SkillName + "-" + time.Now().UTC().Format("20060102150405")

	skill := &models.Skill{
		ID:               local.SkillID,
		DisplayName:      local.SkillName,
		SourceURL:        local.InstallPath,
		RepositoryOwner:  "local",
		RepositoryName:   local.SkillName,
		RepositoryPath:   "",
		CurrentVersionID: versionID,
	}
	if err := a.store.UpsertSkill(ctx, skill); err != nil {
		return err
	}

	// 写入版本记录
	version := &models.SkillVersion{
		ID:             versionID,
		SkillID:        local.SkillID,
		DisplayVersion: "本地",
		GitRef:         "",
		CommitSHA:      "",
		Checksum:       "",
		InstallPath:    local.InstallPath,
	}
	return a.store.InsertVersion(ctx, version)
}

// syncToAgents 将版本目录链接到所有已启用 Agent
func (a *App) syncToAgents(ctx context.Context, versionDir, skillID string) (synced, conflicts []string, err error) {
	// 查询已启用 Agent
	agentList, err := a.store.ListAgents(ctx)
	if err != nil {
		return nil, nil, err
	}

	enabled := make(map[string]string)
	for _, ag := range agentList {
		if ag.Enabled {
			enabled[ag.AdapterID] = ag.SkillsPath
		}
	}
	if len(enabled) == 0 {
		return []string{}, []string{}, nil
	}

	// 取 SkillID 的最后一段作为 Agent 目录中的 Skill 名
	skillName := skillNameFromID(skillID)

	syncer := agents.NewSyncer(platform.NewLinker(), enabled)
	results, err := syncer.SyncSkills(ctx, map[string]string{skillName: versionDir})
	if err != nil {
		return nil, nil, err
	}

	for _, r := range results {
		if r.Status == "linked" {
			synced = append(synced, r.AgentID)
		} else if r.Status == "conflict" {
			conflicts = append(conflicts, r.AgentID+"/"+r.SkillName)
		}
	}
	if synced == nil {
		synced = []string{}
	}
	if conflicts == nil {
		conflicts = []string{}
	}
	return synced, conflicts, nil
}

// displayNameFromRef 从仓库引用构造显示名
func displayNameFromRef(ref *github.RepoRef) string {
	name := ref.Repo
	if p := ref.SubPath; p != "" {
		parts := strings.Split(p, "/")
		name = parts[len(parts)-1]
	}
	return name
}

// skillNameFromID 从 Skill ID（github.com/o/r/name）取最后一段作为目录名
func skillNameFromID(skillID string) string {
	parts := strings.Split(skillID, "/")
	return parts[len(parts)-1]
}

// ---------- 工具 ----------

// newGHClient 创建一个配置了下载镜像和 token 的 GitHub 客户端
func (a *App) newGHClient() *github.Client {
	gh := github.NewClient()
	if a.cfg != nil {
		gh.SetMirror(a.cfg.GetConfig().DownloadMirror)
	}
	if a.ghToken != "" {
		gh.SetToken(a.ghToken)
	}
	return gh
}

// fetchAccountAndCache 用当前 token 获取 GitHub 账号信息并缓存到内存与配置。
// 失败时静默返回，不阻断授权主流程（账号信息获取失败不影响授权本身）。
func (a *App) fetchAccountAndCache() {
	if a.ghToken == "" {
		return
	}
	gh := a.newGHClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	user, err := gh.GetCurrentUser(ctx)
	if err != nil {
		return
	}
	a.ghUserLogin = user.Login
	a.ghUserName = user.Name
	if a.cfg != nil {
		_ = a.cfg.SetGitHubAccount(user.Login, user.Name)
	}
}

// GetGitHubRateLimit 返回当前 GitHub 速率限制（core 资源），供设置页展示剩余额度。
// 已授权时基于当前 token 返回用户额度，未授权返回公共额度。该查询本身不消耗配额。
func (a *App) GetGitHubRateLimit() (map[string]any, error) {
	gh := a.newGHClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	info, err := gh.GetRateLimit(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询剩余额度失败: %w", err)
	}
	resetTime := time.Unix(info.ResetUnix, 0)
	minutes := int(time.Until(resetTime).Minutes())
	if minutes < 0 {
		minutes = 0
	}
	return map[string]any{
		"limit":         info.Limit,
		"remaining":     info.Remaining,
		"resetUnix":     info.ResetUnix,
		"resetTime":     resetTime.Format("15:04"),
		"resetMinutes":  minutes,
		"authenticated": a.ghToken != "",
	}, nil
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// AppError 用于向前端返回用户可读的错误信息
type AppError struct {
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}
