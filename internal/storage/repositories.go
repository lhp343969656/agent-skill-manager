package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"agent-skill-manager/internal/models"
)

const timeLayout = time.RFC3339

// ---------- skills ----------

// ListSkills 返回所有已安装的 Skill
func (s *Store) ListSkills(ctx context.Context) ([]models.Skill, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, s.display_name, s.source_url, s.repository_owner, s.repository_name,
		        s.repository_path, s.current_version_id, s.note, s.created_at, s.updated_at,
		        COALESCE(v.display_version, ''), COALESCE(v.installed_at, '')
		 FROM skills s
		 LEFT JOIN skill_versions v ON v.id = s.current_version_id
		 ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []models.Skill
	for rows.Next() {
		var sk models.Skill
		var created, updated, installed string
		if err := rows.Scan(&sk.ID, &sk.DisplayName, &sk.SourceURL, &sk.RepositoryOwner,
			&sk.RepositoryName, &sk.RepositoryPath, &sk.CurrentVersionID, &sk.Note, &created, &updated,
			&sk.DisplayVersion, &installed); err != nil {
			return nil, err
		}
		sk.CreatedAt = parseTime(created)
		sk.UpdatedAt = parseTime(updated)
		sk.InstalledAt = parseTime(installed)
		skills = append(skills, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if skills == nil {
		skills = []models.Skill{}
	}
	return skills, nil
}

// InsertSkill 插入一条 Skill 记录
func (s *Store) InsertSkill(ctx context.Context, sk *models.Skill) error {
	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO skills (id, display_name, source_url, repository_owner, repository_name,
		        repository_path, current_version_id, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sk.ID, sk.DisplayName, sk.SourceURL, sk.RepositoryOwner, sk.RepositoryName,
		sk.RepositoryPath, sk.CurrentVersionID, sk.Note, now, now)
	return err
}

// GetSkill 按 ID 查询单个 Skill 记录
func (s *Store) GetSkill(ctx context.Context, id string) (*models.Skill, error) {
	var sk models.Skill
	var created, updated string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, display_name, source_url, repository_owner, repository_name,
		        repository_path, current_version_id, note, created_at, updated_at
		 FROM skills WHERE id = ?`, id).
		Scan(&sk.ID, &sk.DisplayName, &sk.SourceURL, &sk.RepositoryOwner,
			&sk.RepositoryName, &sk.RepositoryPath, &sk.CurrentVersionID, &sk.Note, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sk.CreatedAt = parseTime(created)
	sk.UpdatedAt = parseTime(updated)
	return &sk, nil
}

// DeleteSkill 删除一条 Skill 记录及其版本、链接记录（级联删除）
func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE id = ?`, id)
	return err
}

// UpsertSkill 插入或更新一条 Skill 记录（用于覆盖更新场景）
func (s *Store) UpsertSkill(ctx context.Context, sk *models.Skill) error {
	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO skills (id, display_name, source_url, repository_owner, repository_name,
		        repository_path, current_version_id, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   display_name=excluded.display_name,
		   source_url=excluded.source_url,
		   current_version_id=excluded.current_version_id,
		   note=CASE WHEN excluded.note <> '' THEN excluded.note ELSE skills.note END,
		   updated_at=excluded.updated_at`,
		sk.ID, sk.DisplayName, sk.SourceURL, sk.RepositoryOwner, sk.RepositoryName,
		sk.RepositoryPath, sk.CurrentVersionID, sk.Note, now, now)
	return err
}

// UpdateSkillNote 更新某个 Skill 的备注名（别名）。备注为空时不清空已有备注。
func (s *Store) UpdateSkillNote(ctx context.Context, id, note string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE skills SET note = ?, updated_at = ? WHERE id = ?`,
		note, time.Now().UTC().Format(timeLayout), id)
	return err
}

// ---------- skill_versions ----------

// InsertVersion 插入一条 Skill 版本记录
func (s *Store) InsertVersion(ctx context.Context, v *models.SkillVersion) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO skill_versions (id, skill_id, display_version, git_ref, commit_sha,
		        checksum, install_path, installed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.SkillID, v.DisplayVersion, v.GitRef, v.CommitSHA,
		v.Checksum, v.InstallPath, time.Now().UTC().Format(timeLayout))
	return err
}

// ListVersions 返回某个 Skill 的全部版本记录（按安装时间倒序）
func (s *Store) ListVersions(ctx context.Context, skillID string) ([]models.SkillVersion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, display_version, git_ref, commit_sha,
		        checksum, install_path, installed_at
		 FROM skill_versions WHERE skill_id = ? ORDER BY installed_at DESC`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.SkillVersion
	for rows.Next() {
		var v models.SkillVersion
		var installed string
		if err := rows.Scan(&v.ID, &v.SkillID, &v.DisplayVersion, &v.GitRef,
			&v.CommitSHA, &v.Checksum, &v.InstallPath, &installed); err != nil {
			return nil, err
		}
		v.InstalledAt = parseTime(installed)
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if versions == nil {
		versions = []models.SkillVersion{}
	}
	return versions, nil
}

// GetVersion 按版本 ID 查询单个版本记录
func (s *Store) GetVersion(ctx context.Context, versionID string) (*models.SkillVersion, error) {
	var v models.SkillVersion
	var installed string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, skill_id, display_version, git_ref, commit_sha,
		        checksum, install_path, installed_at
		 FROM skill_versions WHERE id = ?`, versionID).
		Scan(&v.ID, &v.SkillID, &v.DisplayVersion, &v.GitRef,
			&v.CommitSHA, &v.Checksum, &v.InstallPath, &installed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.InstalledAt = parseTime(installed)
	return &v, nil
}

// ---------- agents ----------

// ListAgents 返回所有 Agent
func (s *Store) ListAgents(ctx context.Context) ([]models.Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, adapter_id, display_name, skills_path, enabled, detected, updated_at
		 FROM agents ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		var updated string
		var enabled, detected int
		if err := rows.Scan(&a.ID, &a.AdapterID, &a.DisplayName, &a.SkillsPath,
			&enabled, &detected, &updated); err != nil {
			return nil, err
		}
		a.Enabled = enabled != 0
		a.Detected = detected != 0
		a.UpdatedAt = parseTime(updated)
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if agents == nil {
		agents = []models.Agent{}
	}
	return agents, nil
}

// UpsertAgent 插入或更新一条 Agent 记录
func (s *Store) UpsertAgent(ctx context.Context, a *models.Agent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agents (id, adapter_id, display_name, skills_path, enabled, detected, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   display_name=excluded.display_name,
		   skills_path=excluded.skills_path,
		   enabled=excluded.enabled,
		   detected=excluded.detected,
		   updated_at=excluded.updated_at`,
		a.ID, a.AdapterID, a.DisplayName, a.SkillsPath,
		boolToInt(a.Enabled), boolToInt(a.Detected), time.Now().UTC().Format(timeLayout))
	return err
}

// ---------- managed_links ----------

// InsertLink 插入一条管理链接记录
func (s *Store) InsertLink(ctx context.Context, l *models.ManagedLink) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO managed_links (id, agent_id, skill_id, skill_version_id,
		        source_path, target_path, link_mode, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.AgentID, l.SkillID, l.SkillVersionID,
		l.SourcePath, l.TargetPath, l.LinkMode, l.Status,
		time.Now().UTC().Format(timeLayout))
	return err
}

// ---------- helpers ----------

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ensureNotNull 供调试使用：检查数据库连接是否可用
func (s *Store) ensureNotNull(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	return nil
}
