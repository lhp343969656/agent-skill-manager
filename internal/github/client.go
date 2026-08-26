package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-skill-manager/internal/netproxy"

	"github.com/google/go-github/v69/github"
)

// Client 封装 GitHub API 访问
type Client struct {
	api    *github.Client
	http   *http.Client
	mirror string // 可选 GitHub 下载加速镜像前缀
	token  string // 可选 GitHub token（提升 API 配额）
}

// NewClient 创建 GitHub 客户端（无认证，访问公开仓库）。
// 使用跟随系统代理的 HTTP 客户端，确保能访问 GitHub。
func NewClient() *Client {
	httpClient := &http.Client{
		Transport: netproxy.NewTransport(),
		Timeout:   90 * time.Second,
	}
	return &Client{
		api:  github.NewClient(httpClient),
		http: httpClient,
	}
}

// SetMirror 设置 GitHub 下载加速镜像前缀（如 https://ghproxy.com/）。
// 设置后，Archive 下载 URL 会自动改写为镜像加速地址。
func (c *Client) SetMirror(mirror string) {
	c.mirror = strings.TrimSpace(mirror)
}

// SetToken 设置 GitHub token 以提升 API 配额（认证后配额更高）。
// 设置后客户端的 API 请求会携带该 token。
func (c *Client) SetToken(token string) {
	c.token = strings.TrimSpace(token)
	if c.token != "" {
		c.api = c.api.WithAuthToken(c.token)
	}
}

// applyMirror 若配置了镜像，将下载 URL 改写为 <mirror>/<原URL> 形式
func (c *Client) applyMirror(rawURL string) string {
	if c.mirror == "" {
		return rawURL
	}
	return c.mirror + rawURL
}

// RepoInfo 描述仓库的基础信息
type RepoInfo struct {
	DefaultBranch string
	Description   string
	Stars         int
}

// GetRepoInfo 查询仓库信息
func (c *Client) GetRepoInfo(ctx context.Context, ref *RepoRef) (*RepoInfo, error) {
	repo, _, err := c.api.Repositories.Get(ctx, ref.Owner, ref.Repo)
	if err != nil {
		return nil, fmt.Errorf("查询仓库失败: %w", err)
	}
	defaultBranch := repo.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	return &RepoInfo{
		DefaultBranch: defaultBranch,
		Description:   repo.GetDescription(),
		Stars:         repo.GetStargazersCount(),
	}, nil
}

// ResolveRef 确定要使用的 ref（分支/tag/commit）。
// 优先级：URL 中指定 ref > 默认分支。
// 返回精确定位用的 ref 名和默认分支（用于仓库根下载）。
func (c *Client) ResolveRef(ctx context.Context, ref *RepoRef) (string, error) {
	if ref.Ref != "" {
		return ref.Ref, nil
	}
	info, err := c.GetRepoInfo(ctx, ref)
	if err != nil {
		return "", err
	}
	return info.DefaultBranch, nil
}

// ResolveCommitSHA 将 gitRef（分支/tag/commit）解析为精确的 commit SHA。
// 分支和 tag 会解析到对应的 commit；已是完整 SHA 则原样返回。
func (c *Client) ResolveCommitSHA(ctx context.Context, ref *RepoRef, gitRef string) (string, error) {
	commit, _, err := c.api.Repositories.GetCommit(ctx, ref.Owner, ref.Repo, gitRef, nil)
	if err != nil {
		return "", fmt.Errorf("解析 commit SHA 失败: %w", err)
	}
	return commit.GetSHA(), nil
}

// CommitInfo 描述一个提交的信息
type CommitInfo struct {
	SHA     string
	Message string
}

// GetCommitInfo 获取指定 gitRef（分支/tag/commit）对应的提交信息，含 SHA 与提交说明。
// 用于检查更新时向用户展示更新的内容。
func (c *Client) GetCommitInfo(ctx context.Context, ref *RepoRef, gitRef string) (*CommitInfo, error) {
	commit, _, err := c.api.Repositories.GetCommit(ctx, ref.Owner, ref.Repo, gitRef, nil)
	if err != nil {
		return nil, fmt.Errorf("获取提交信息失败: %w", err)
	}
	return &CommitInfo{
		SHA:     commit.GetSHA(),
		Message: commit.GetCommit().GetMessage(),
	}, nil
}

// ArchiveURL 返回仓库 Archive ZIP 的下载地址。
// subPath 为空时下载整个仓库；否则调用 GitHub 的目录下载 API。
// 若配置了镜像前缀，返回的地址会自动改写为镜像加速地址。
func (c *Client) ArchiveURL(ctx context.Context, ref *RepoRef, gitRef string) (string, error) {
	var rawURL string
	if ref.SubPath == "" {
		// 整个仓库：https://codeload.github.com/{owner}/{repo}/zip/{ref}
		rawURL = fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s",
			ref.Owner, ref.Repo, gitRef)
	} else {
		// 子目录：通过 GitHub API 获取下载 URL
		u, _, err := c.api.Repositories.GetArchiveLink(ctx, ref.Owner, ref.Repo,
			github.Zipball, &github.RepositoryContentGetOptions{Ref: gitRef}, 3)
		if err != nil {
			return "", fmt.Errorf("获取下载链接失败: %w", err)
		}
		rawURL = u.String()
	}

	return c.applyMirror(rawURL), nil
}

// Version 描述一个可用版本（Release/Tag/Commit）
type Version struct {
	Kind    string // release | tag | commit
	Display string // 显示名（版本号或 tag）
	Ref     string // git ref（用于下载）
	SHA     string // commit SHA
}

// ListVersions 返回仓库的版本列表，按优先级排序：Release > Tag > Commit
func (c *Client) ListVersions(ctx context.Context, ref *RepoRef, limit int) ([]Version, error) {
	if limit <= 0 {
		limit = 20
	}

	var versions []Version

	// Release（按版本号语义）
	releases, _, err := c.api.Repositories.ListReleases(ctx, ref.Owner, ref.Repo,
		&github.ListOptions{PerPage: limit})
	if err == nil {
		for _, r := range releases {
			versions = append(versions, Version{
				Kind:    "release",
				Display: r.GetTagName(),
				Ref:     r.GetTagName(),
			})
		}
	}

	// Tag
	tags, _, err := c.api.Repositories.ListTags(ctx, ref.Owner, ref.Repo,
		&github.ListOptions{PerPage: limit})
	if err == nil {
		seen := map[string]bool{}
		for _, t := range tags {
			name := t.GetName()
			if seen[name] {
				continue
			}
			seen[name] = true
			versions = append(versions, Version{
				Kind:    "tag",
				Display: name,
				Ref:     name,
			})
		}
	}

	// Commit（默认分支最新）
	if len(versions) == 0 {
		defaultBranch, err := c.ResolveRef(ctx, ref)
		if err == nil {
			commit, _, err := c.api.Repositories.GetCommit(ctx, ref.Owner, ref.Repo, defaultBranch, nil)
			if err == nil {
				versions = append(versions, Version{
					Kind:    "commit",
					Display: commit.GetSHA(),
					Ref:     commit.GetSHA(),
					SHA:     commit.GetSHA(),
				})
			}
		}
	}

	// 去重（同一 ref 只保留一次，release 优先）
	var result []Version
	seenRef := map[string]bool{}
	for _, v := range versions {
		key := strings.ToLower(v.Ref)
		if seenRef[key] {
			continue
		}
		seenRef[key] = true
		result = append(result, v)
	}
	return result, nil
}

// CurrentUser 描述当前认证的 GitHub 用户信息
type CurrentUser struct {
	Login string
	Name  string
}

// GetCurrentUser 返回当前已认证的 GitHub 用户信息（需已设置 token）。
// 使用空字符串 user 表示获取当前认证用户。
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	if c.token == "" {
		return nil, fmt.Errorf("未设置 token，无法获取用户信息")
	}
	u, _, err := c.api.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("获取 GitHub 用户失败: %w", err)
	}
	return &CurrentUser{
		Login: u.GetLogin(),
		Name:  u.GetName(),
	}, nil
}

// LatestTag 返回仓库最新的 tag 名（优先 Release 的 tag；无 tag 返回空字符串）。
// tags 列表按 GitHub API 返回，通常按创建/提交时间排序。
func (c *Client) LatestTag(ctx context.Context, ref *RepoRef) (string, error) {
	// 优先 Release（其 tag 语义更明确）
	releases, _, err := c.api.Repositories.ListReleases(ctx, ref.Owner, ref.Repo,
		&github.ListOptions{PerPage: 1})
	if err == nil && len(releases) > 0 {
		tag := releases[0].GetTagName()
		if tag != "" {
			return tag, nil
		}
	}

	// 其次 Tags
	tags, _, err := c.api.Repositories.ListTags(ctx, ref.Owner, ref.Repo,
		&github.ListOptions{PerPage: 1})
	if err == nil && len(tags) > 0 {
		return tags[0].GetName(), nil
	}

	return "", nil
}

// RateLimitInfo 描述 GitHub 速率限制信息（core 资源，用于设置页展示剩余额度）。
type RateLimitInfo struct {
	Limit      int   // 每小时的总额度
	Remaining  int   // 当前剩余额度
	ResetUnix  int64 // 额度重置的 Unix 时间戳（秒）
}

// GetRateLimit 查询当前 GitHub 速率限制（core 资源）。
// 该接口本身不消耗配额。带 token 时返回已授权用户的额度，否则返回匿名公共额度。
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimitInfo, error) {
	rl, _, err := c.api.RateLimit.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询速率限制失败: %w", err)
	}
	if rl.Core == nil {
		return nil, fmt.Errorf("查询速率限制失败: 返回数据为空")
	}
	return &RateLimitInfo{
		Limit:     rl.Core.Limit,
		Remaining: rl.Core.Remaining,
		ResetUnix: rl.Core.Reset.Time.Unix(),
	}, nil
}
