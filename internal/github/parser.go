package github

import (
	"fmt"
	"strings"
)

// RepoRef 描述一个 GitHub 仓库及其子目录
type RepoRef struct {
	Owner    string // 仓库所有者
	Repo     string // 仓库名
	SubPath  string // 仓库内子目录路径（可为空）
	Ref      string // 可选：分支/tag/commit
	Original string // 原始 URL
}

// ParseURL 解析 GitHub 仓库 URL。
// 支持格式：
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo/tree/main
//   - https://github.com/owner/repo/tree/main/sub/dir
//   - https://github.com/owner/repo/blob/main/sub/dir/SKILL.md
//   - github.com/owner/repo（无协议前缀）
func ParseURL(raw string) (*RepoRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}

	ref := &RepoRef{Original: raw}

	// 去掉协议前缀和前缀斜杠
	u := raw
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "www.")
	u = strings.TrimPrefix(u, "github.com/")
	u = strings.Trim(u, "/")

	// 现在 u 形如 owner/repo[/tree/...|/blob/...]
	parts := strings.Split(u, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无法识别的 GitHub 链接: %s", raw)
	}

	ref.Owner = parts[0]
	ref.Repo = parts[1]

	// 后续部分是 /tree/<ref>/... 或 /blob/<ref>/...
	if len(parts) > 2 {
		kind := parts[2]
		rest := parts[3:]

		switch kind {
		case "tree", "blob":
			if len(rest) == 0 {
				return nil, fmt.Errorf("链接缺少分支/标签信息: %s", raw)
			}
			ref.Ref = rest[0]
			// 去掉文件名（blob 指向文件时）
			if kind == "blob" && len(rest) > 1 {
				// blob 指向单个文件，子目录取文件所在目录
				rest = rest[:len(rest)-1]
			}
			if len(rest) > 1 {
				ref.SubPath = strings.Join(rest[1:], "/")
			}
		case "commit":
			if len(rest) > 0 {
				ref.Ref = rest[0]
			}
			if len(rest) > 1 {
				ref.SubPath = strings.Join(rest[1:], "/")
			}
		default:
			// 形如 owner/repo/直接子路径（无 tree 前缀），视为子目录
			ref.SubPath = strings.Join(parts[2:], "/")
		}
	}

	if ref.Owner == "" || ref.Repo == "" {
		return nil, fmt.Errorf("无法识别的 GitHub 链接: %s", raw)
	}

	return ref, nil
}

// RepoURL 返回仓库的根 URL
func (r *RepoRef) RepoURL() string {
	return fmt.Sprintf("https://github.com/%s/%s", r.Owner, r.Repo)
}

// CleanSubPath 去除子目录首尾斜杠
func (r *RepoRef) CleanSubPath() string {
	return strings.Trim(r.SubPath, "/")
}
