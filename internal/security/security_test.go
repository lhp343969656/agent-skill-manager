package security

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// makeZip 创建一个 zip 文件
func makeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, content := range entries {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	f.Close()
	return zipPath
}

func TestExtractZipNormal(t *testing.T) {
	zipPath := makeZip(t, map[string]string{
		"repo-main/SKILL.md":       "# Test Skill",
		"repo-main/scripts/x.sh":   "echo hi",
		"repo-main/skills/a/a.md":  "content a",
	})
	dest := filepath.Join(t.TempDir(), "extract")

	if err := ExtractZip(zipPath, dest); err != nil {
		t.Fatalf("解压失败: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "repo-main", "SKILL.md")); err != nil {
		t.Errorf("SKILL.md 未解压成功: %v", err)
	}
}

func TestExtractZipZipSlip(t *testing.T) {
	// 构造恶意 zip：条目包含 ../ 路径穿越
	zipPath := makeZip(t, map[string]string{
		"../../../../evil.txt": "pwned",
	})
	dest := filepath.Join(t.TempDir(), "extract")

	err := ExtractZip(zipPath, dest)
	if err == nil {
		t.Fatal("期望检测到 Zip Slip，实际没有")
	}

	// 确保没有文件被解压到 dest 之外
	if _, err := os.Stat(filepath.Join(dest, "..", "..", "..", "..", "evil.txt")); !os.IsNotExist(err) {
		t.Errorf("恶意文件不应被写入: %v", err)
	}
}

func TestExtractZipAbsolutePath(t *testing.T) {
	// 绝对路径条目
	zipPath := makeZip(t, map[string]string{
		"/tmp/evil.txt": "pwned",
	})
	dest := filepath.Join(t.TempDir(), "extract")

	err := ExtractZip(zipPath, dest)
	if err == nil {
		t.Fatal("期望检测到绝对路径条目，实际没有")
	}
}

func TestSHA256File(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	os.WriteFile(file, []byte("hello"), 0o644)

	sum, err := SHA256(file)
	if err != nil {
		t.Fatal(err)
	}
	// SHA-256 of "hello"
	if sum != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("SHA-256 不正确: %s", sum)
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	os.WriteFile(file, []byte("hello"), 0o644)

	sum, _ := SHA256(file)
	ok, err := Verify(file, sum)
	if err != nil || !ok {
		t.Errorf("校验应通过: ok=%v err=%v", ok, err)
	}

	ok, _ = Verify(file, "wrong")
	if ok {
		t.Error("错误哈希不应校验通过")
	}
}
