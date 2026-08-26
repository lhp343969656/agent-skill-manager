package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store 封装 SQLite 数据库连接
type Store struct {
	db *sql.DB
}

// Open 打开或创建位于共享目录 .manager/state.db 的数据库。
// 目录不存在时会自动创建。
func Open(sharedDir string) (*Store, error) {
	managerDir := filepath.Join(sharedDir, ".manager")
	if err := os.MkdirAll(managerDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建管理器目录失败: %w", err)
	}

	// 创建缓存、锁、日志子目录
	for _, sub := range []string{"cache", "locks", "logs"} {
		if err := os.MkdirAll(filepath.Join(managerDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("创建目录 %s 失败: %w", sub, err)
		}
	}

	dbPath := filepath.Join(managerDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 启用 WAL 模式提高并发读写安全性
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("启用外键失败: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// migrate 创建核心表结构（设计文档第 12 节）
func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skills (
			id                  TEXT PRIMARY KEY,
			display_name        TEXT NOT NULL,
			source_url          TEXT NOT NULL,
			repository_owner    TEXT NOT NULL,
			repository_name     TEXT NOT NULL,
			repository_path     TEXT NOT NULL DEFAULT '',
			current_version_id  TEXT,
			note                TEXT NOT NULL DEFAULT '',
			created_at          TEXT NOT NULL,
			updated_at          TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS skill_versions (
			id               TEXT PRIMARY KEY,
			skill_id         TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
			display_version  TEXT,
			git_ref          TEXT NOT NULL,
			commit_sha       TEXT NOT NULL,
			checksum         TEXT NOT NULL,
			install_path     TEXT NOT NULL,
			installed_at     TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agents (
			id           TEXT PRIMARY KEY,
			adapter_id   TEXT NOT NULL,
			display_name TEXT NOT NULL,
			skills_path  TEXT NOT NULL,
			enabled      INTEGER NOT NULL DEFAULT 0,
			detected     INTEGER NOT NULL DEFAULT 0,
			updated_at   TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS managed_links (
			id               TEXT PRIMARY KEY,
			agent_id         TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			skill_id         TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
			skill_version_id TEXT NOT NULL REFERENCES skill_versions(id) ON DELETE CASCADE,
			source_path      TEXT NOT NULL,
			target_path      TEXT NOT NULL,
			link_mode        TEXT NOT NULL,
			status           TEXT NOT NULL,
			created_at       TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_skill_versions_skill_id ON skill_versions(skill_id);`,
		`CREATE INDEX IF NOT EXISTS idx_managed_links_agent_id ON managed_links(agent_id);`,
		`CREATE INDEX IF NOT EXISTS idx_managed_links_skill_id ON managed_links(skill_id);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("执行迁移语句失败: %w", err)
		}
	}

	// 兼容旧数据库：skills 表可能缺少 note 列（备注名），需补充
	var hasNote int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('skills') WHERE name='note'`).Scan(&hasNote); err != nil {
		return fmt.Errorf("检查 note 列失败: %w", err)
	}
	if hasNote == 0 {
		if _, err := s.db.Exec(`ALTER TABLE skills ADD COLUMN note TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("添加 note 列失败: %w", err)
		}
	}
	return nil
}

// DB 返回底层数据库连接（供业务层使用）
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	return s.db.Close()
}
