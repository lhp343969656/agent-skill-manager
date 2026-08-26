package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-skill-manager/internal/skills"
)

// LocalInstallResult 描述一次本地安装的结果
type LocalInstallResult struct {
	// SkillID 是本地 Skill 的唯一标识（local/<name>）
	SkillID string
	// SkillName 是 Agent 目录中使用的 Skill 名
	SkillName string
	// InstallPath 是共享目录中的实际安装路径
	InstallPath string
}

// InstallLocal 从本地路径安装 Skill。
// path 可以是：
//   - 一个文件夹（扫描其下所有 SKILL.md，每个父目录作为一个 Skill）
//   - 一个 SKILL.md 文件（其所在目录作为一个 Skill）
//
// 本地 Skill 采用覆盖更新策略：同名 Skill 重新安装会覆盖。
func (i *Installer) InstallLocal(ctx context.Context, path string) ([]LocalInstallResult, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("无法解析路径: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("无法访问本地路径: %w", err)
	}

	// 收集要安装的 Skill 源目录
	var skillDirs []string

	if info.IsDir() {
		// 文件夹：扫描所有 SKILL.md
		scanner := skills.NewScanner()
		files, err := scanner.Scan(abs)
		if err != nil {
			return nil, fmt.Errorf("扫描本地文件夹失败: %w", err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("所选文件夹中未找到 SKILL.md")
		}
		for _, f := range files {
			skillDirs = append(skillDirs, f.Dir)
		}
	} else {
		// 单个文件：必须命名为 SKILL.md
		if !strings.EqualFold(filepath.Base(abs), "SKILL.md") {
			return nil, fmt.Errorf("本地文件必须是 SKILL.md（您选择了: %s）", filepath.Base(abs))
		}
		skillDirs = append(skillDirs, filepath.Dir(abs))
	}

	// 逐个安装
	var results []LocalInstallResult
	for _, srcDir := range skillDirs {
		result, err := i.installLocalSkill(srcDir)
		if err != nil {
			return results, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// installLocalSkill 将单个 Skill 源目录复制到共享目录
func (i *Installer) installLocalSkill(srcDir string) (*LocalInstallResult, error) {
	// Skill 名取源目录名
	skillName := filepath.Base(srcDir)

	// 目标目录：sharedDir/local/<skillName>
	targetDir := filepath.Join(i.sharedDir, "local", skillName)

	// 覆盖更新：先删除旧目录再复制
	if err := os.RemoveAll(targetDir); err != nil {
		return nil, fmt.Errorf("清理旧版本失败: %w", err)
	}
	if err := filepathWalkCopy(srcDir, targetDir); err != nil {
		return nil, fmt.Errorf("复制 Skill 失败: %w", err)
	}

	return &LocalInstallResult{
		SkillID:     "local/" + skillName,
		SkillName:   skillName,
		InstallPath: targetDir,
	}, nil
}
