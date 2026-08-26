package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SkillFile 描述一个被扫描到的 SKILL.md
type SkillFile struct {
	// Path 是 SKILL.md 相对于内容根目录的路径
	RelPath string
	// Dir 是 SKILL.md 所在目录的绝对路径
	Dir string
	// Name 是 SKILL.md 的绝对路径
	Path string
	// Content 是 SKILL.md 的原始内容
	Content string
}

// Scanner 扫描目录中的 SKILL.md 文件
type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan 在 root 目录中递归查找所有 SKILL.md（大小写不敏感）。
// 返回找到的 SkillFile 列表；未找到时返回空列表（不报错）。
func (s *Scanner) Scan(root string) ([]SkillFile, error) {
	var results []SkillFile

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// 跳过隐藏目录和常见无关目录
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.EqualFold(d.Name(), "SKILL.md") {
			return nil
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("读取 %s 失败: %w", path, err)
		}

		rel, _ := filepath.Rel(root, path)
		results = append(results, SkillFile{
			RelPath: filepath.ToSlash(rel),
			Dir:     filepath.Dir(abs),
			Path:    abs,
			Content: string(content),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}
	return results, nil
}
