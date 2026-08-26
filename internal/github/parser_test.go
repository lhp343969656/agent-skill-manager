package github

import "testing"

func TestApplyMirror(t *testing.T) {
	// 未配置镜像时返回原 URL
	c := NewClient()
	if url := c.applyMirror("https://codeload.github.com/o/r/zip/main"); url != "https://codeload.github.com/o/r/zip/main" {
		t.Errorf("无镜像时应返回原 URL: %s", url)
	}

	// 配置镜像后改写为 <mirror>/<原URL>
	c.SetMirror("https://ghproxy.com/")
	url := c.applyMirror("https://codeload.github.com/o/r/zip/main")
	want := "https://ghproxy.com/https://codeload.github.com/o/r/zip/main"
	if url != want {
		t.Errorf("镜像改写错误:\n got %s\nwant %s", url, want)
	}

	// 带空格的镜像应被清理
	c.SetMirror("  https://ghproxy.com/  ")
	if url := c.applyMirror("https://github.com/o/r"); url != "https://ghproxy.com/https://github.com/o/r" {
		t.Errorf("镜像应清理空白: %s", url)
	}
}

func TestParseURL(t *testing.T) {
	cases := []struct {
		raw     string
		owner   string
		repo    string
		subPath string
		ref     string
		wantErr bool
	}{
		{"https://github.com/owner/repo", "owner", "repo", "", "", false},
		{"https://github.com/owner/repo/tree/main", "owner", "repo", "", "main", false},
		{"https://github.com/owner/repo/tree/main/skills/abc", "owner", "repo", "skills/abc", "main", false},
		{"https://github.com/owner/repo/blob/main/skills/abc/SKILL.md", "owner", "repo", "skills/abc", "main", false},
		{"github.com/owner/repo", "owner", "repo", "", "", false},
		{"https://github.com/owner/repo/tree/main/sub/dir/deep", "owner", "repo", "sub/dir/deep", "main", false},
		{"https://github.com/owner/repo/tree/v1.0.0", "owner", "repo", "", "v1.0.0", false},
		{"https://github.com/owner/repo/commit/abc123", "owner", "repo", "", "abc123", false},
		{"", "", "", "", "", true},
		{"not-a-url", "", "", "", "", true},
	}

	for _, c := range cases {
		ref, err := ParseURL(c.raw)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseURL(%q) 期望错误，实际成功: %+v", c.raw, ref)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseURL(%q) 失败: %v", c.raw, err)
			continue
		}
		if ref.Owner != c.owner || ref.Repo != c.repo || ref.SubPath != c.subPath || ref.Ref != c.ref {
			t.Errorf("ParseURL(%q) = owner=%s repo=%s sub=%q ref=%q, 期望 owner=%s repo=%s sub=%q ref=%q",
				c.raw, ref.Owner, ref.Repo, ref.SubPath, ref.Ref,
				c.owner, c.repo, c.subPath, c.ref)
		}
	}
}
