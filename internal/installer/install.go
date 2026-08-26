package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"agent-skill-manager/internal/github"
	"agent-skill-manager/internal/security"
	"agent-skill-manager/internal/skills"
)

// Installer 负责安装 Skill 到共享目录的不可变版本目录
type Installer struct {
	sharedDir string
	ghClient  *github.Client
}

// New 创建安装器
func New(sharedDir string) *Installer {
	return &Installer{
		sharedDir: sharedDir,
		ghClient:  github.NewClient(),
	}
}

// InstallOptions 描述一次安装请求
type InstallOptions struct {
	RepoRef *github.RepoRef
	GitRef  string // 解析后的分支/tag/commit
	Commit  string // 精确 commit SHA
	// SkillRelPath 是仓库内要安装的单个技能的相对路径（相对于仓库根）。
	// 为空时回退到旧行为：取扫描到的第一个技能。
	SkillRelPath string
}

// InstallResult 描述安装结果
type InstallResult struct {
	SkillID     string
	InstallPath string
	// SkillName 是安装的技能名（技能目录名），供前端展示与 Agent 目录命名
	SkillName string
}

// Install 执行完整安装流程：
// 下载 -> 安全解压 -> 扫描 SKILL.md -> 写入不可变版本目录
func (i *Installer) Install(ctx context.Context, opt InstallOptions) (*InstallResult, error) {
	// 1. 确定 Archive 下载 URL
	archiveURL, err := i.ghClient.ArchiveURL(ctx, opt.RepoRef, opt.GitRef)
	if err != nil {
		return nil, err
	}

	// 2. 下载并解压到临时目录
	tmpDir, err := os.MkdirTemp("", "skill-extract-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := security.DownloadAndExtract(archiveURL, tmpDir); err != nil {
		return nil, err
	}

	// 3. 找到内容根目录（GitHub zip 含顶层目录）
	root, err := security.FindRootDir(tmpDir)
	if err != nil {
		return nil, err
	}

	// 4. 扫描 SKILL.md
	scanner := skills.NewScanner()
	skillFiles, err := scanner.Scan(root)
	if err != nil {
		return nil, err
	}
	if len(skillFiles) == 0 {
		return nil, fmt.Errorf("未在仓库中找到 SKILL.md")
	}

	// 5. 定位要安装的技能目录（默认取第一个；也可指定仓库内相对路径）
	var skillFile *skills.SkillFile
	if rel := cleanPath(opt.SkillRelPath); rel != "" {
		found := false
		for i := range skillFiles {
			if sameDirRel(skillFiles[i], root, rel) {
				skillFile = &skillFiles[i]
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("在仓库中未找到指定技能: %s", opt.SkillRelPath)
		}
	} else {
		skillFile = &skillFiles[0]
	}

	// 6. 确定 Skill 身份与版本目录。
	// 使用技能自身的目录名作为安装目录名，使同一仓库内的多个技能各自独立。
	skillName := filepath.Base(skillFile.Dir)
	skillID := buildSkillID(opt.RepoRef, skillName)
	versionDir := filepath.Join(packagesRoot(i.sharedDir), skillPath(skillID), commitDir(opt.Commit))
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建版本目录失败: %w", err)
	}

	// 7. 将 SKILL.md 所在目录复制到版本目录（该目录即 Skill 内容）
	if err := copySkillDir(skillFile.Dir, versionDir); err != nil {
		return nil, err
	}

	return &InstallResult{
		SkillID:     skillID,
		InstallPath: versionDir,
		SkillName:   skillName,
	}, nil
}

// sameDirRel 判断某个 SKILL.md 所在的目录，与仓库内的相对路径 rel 是否一致
func sameDirRel(sf skills.SkillFile, root, rel string) bool {
	dirRel, err := filepath.Rel(root, sf.Dir)
	if err != nil {
		return false
	}
	return cleanPath(filepath.ToSlash(dirRel)) == rel
}

// buildSkillID 构造 Skill 唯一 ID：github.com/owner/repo[/subpath]/skillName
// 加入技能名，使同一仓库内的多个技能拥有各自独立的 ID。
func buildSkillID(ref *github.RepoRef, skillName string) string {
	base := "github.com/" + ref.Owner + "/" + ref.Repo
	if p := cleanPath(ref.SubPath); p != "" {
		base += "/" + p
	}
	if skillName != "" {
		base += "/" + skillName
	}
	return base
}

// skillPath 将 Skill ID 转为相对路径
func skillPath(id string) string {
	return filepath.FromSlash(id)
}

// commitDir 返回版本目录名（commit SHA）
func commitDir(commit string) string {
	if commit == "" {
		return "latest"
	}
	return commit
}

// packagesRoot 返回共享目录下 packages 根路径
func packagesRoot(sharedDir string) string {
	return filepath.Join(sharedDir, "packages")
}

// copySkillDir 递归复制 Skill 内容目录到目标版本目录
func copySkillDir(src, dst string) error {
	return filepathWalkCopy(src, dst)
}

// cleanPath 去除首尾斜杠
func cleanPath(p string) string {
	return trimSlashes(p)
}

func trimSlashes(p string) string {
	for len(p) > 0 && (p[0] == '/' || p[0] == '\\') {
		p = p[1:]
	}
	for len(p) > 0 && (p[len(p)-1] == '/' || p[len(p)-1] == '\\') {
		p = p[:len(p)-1]
	}
	return p
}
