package security

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"agent-skill-manager/internal/netproxy"
)

// 解压限制
const (
	// MaxArchiveBytes 压缩包最大体积（200MB）
	MaxArchiveBytes = 200 << 20
	// MaxExtractedBytes 解压后总字节上限（500MB）
	MaxExtractedBytes = 500 << 20
	// MaxExtractedFiles 解压后文件数量上限
	MaxExtractedFiles = 20000
)

// HTTPClient 用于下载，可在测试中替换。默认跟随系统代理，确保能访问 GitHub。
var HTTPClient = &http.Client{Transport: netproxy.NewTransport(), Timeout: 120 * time.Second}

// DownloadAndExtract 从 url 下载 ZIP 并安全解压到目标目录。
func DownloadAndExtract(url, destDir string) error {
	// 下载到临时文件
	tmpFile, err := os.CreateTemp("", "skill-download-*.zip")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if err := downloadTo(url, tmpFile); err != nil {
		return err
	}

	// 解压
	return ExtractZip(tmpPath, destDir)
}

// downloadTo 下载 url 内容写入 w，限制大小
func downloadTo(url string, w io.Writer) error {
	resp, err := HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回状态码 %d", resp.StatusCode)
	}

	// 限制大小
	limited := io.LimitReader(resp.Body, MaxArchiveBytes)
	n, err := io.Copy(w, limited)
	if err != nil {
		return fmt.Errorf("写入下载内容失败: %w", err)
	}
	if n >= MaxArchiveBytes {
		return fmt.Errorf("下载内容超过大小限制 (%dMB)", MaxArchiveBytes>>20)
	}
	return nil
}

// ExtractZip 安全解压 zip 文件到 destDir。
// 防 Zip Slip、限制文件数量和总体积。
func ExtractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开压缩包失败: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	var totalBytes int64
	fileCount := 0

	for _, file := range reader.File {
		// 统计文件数量
		if !file.FileInfo().IsDir() {
			fileCount++
			if fileCount > MaxExtractedFiles {
				return fmt.Errorf("压缩包文件数量超过限制 (%d)", MaxExtractedFiles)
			}
		}

		// 防 Zip Slip
		target, err := SanitizeEntryPath(destDir, file.Name)
		if err != nil {
			return err
		}

		// 累计解压大小
		totalBytes += int64(file.UncompressedSize64)
		if totalBytes > MaxExtractedBytes {
			return fmt.Errorf("解压内容超过大小限制 (%dMB)", MaxExtractedBytes>>20)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		// 解压文件
		if err := extractFile(file, target); err != nil {
			return err
		}
	}

	return nil
}

// extractFile 解压单个 zip 文件条目
func extractFile(file *zip.File, target string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()

	limited := io.LimitReader(src, MaxExtractedBytes)
	if _, err := io.Copy(dst, limited); err != nil {
		return err
	}
	return nil
}

// FindRootDir 在解压目录中查找内容根目录。
// GitHub Archive ZIP 通常包含一个形如 {repo}-{ref}/ 的顶层目录。
func FindRootDir(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return extractDir, nil
	}
	// 只有一个顶层目录时返回它
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}
	return extractDir, nil
}
