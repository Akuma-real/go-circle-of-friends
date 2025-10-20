// 包 store 提供存储实现（SQLite），包含表迁移/写入/查询/清理等操作。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"go-circle-of-friends/internal/model"
)

// SQLite 封装 *sql.DB，基于 modernc.org/sqlite（纯 Go 实现）。
type SQLite struct {
	db *sql.DB
}

// OpenSQLite 打开 SQLite 数据库并执行自动迁移。
func OpenSQLite(path string) (*SQLite, error) {
	// 说明：modernc sqlite 的 DSN 可直接使用文件路径，或以 'file:...' 前缀表示
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

// Reset 清空业务数据表（不删除数据库文件）。
func (s *SQLite) Reset(ctx context.Context) error {
	// 顺序：先清 posts 再清 friends，避免潜在外键依赖（当前无外键，仅为稳妥）
	if _, err := s.db.ExecContext(ctx, `DELETE FROM posts`); err != nil {
		return fmt.Errorf("delete posts: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM friends`); err != nil {
		return fmt.Errorf("delete friends: %w", err)
	}
	return nil
}

// migrate 执行建表语句，保持幂等。
func (s *SQLite) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS friends (
            name TEXT,
            link TEXT UNIQUE,
            avatar TEXT,
            error TEXT,
            created_at TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS posts (
            title TEXT,
            created TIMESTAMP,
            updated TIMESTAMP,
            link TEXT UNIQUE,
            author TEXT,
            avatar TEXT,
            rule TEXT,
            created_at TIMESTAMP
        );`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec migrate: %w", err)
		}
	}
	return nil
}

// UpsertFriend 插入或更新朋友信息（link 唯一约束）。
func (s *SQLite) UpsertFriend(ctx context.Context, f model.Friend) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO friends(name, link, avatar, error, created_at)
        VALUES(?,?,?,?,?)
        ON CONFLICT(link) DO UPDATE SET name=excluded.name, avatar=excluded.avatar, error=excluded.error`,
		f.Name, f.Link, f.Avatar, f.Error, nowOr(f.CreatedAt))
	if err != nil {
		return fmt.Errorf("upsert friend %s: %w", f.Link, err)
	}
	return nil
}

// UpsertPost 插入或更新文章（link 唯一约束）。
func (s *SQLite) UpsertPost(ctx context.Context, p model.Post) error {
	if p.Link == "" {
		return errors.New("post.link required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO posts(title, created, updated, link, author, avatar, rule, created_at)
        VALUES(?,?,?,?,?,?,?,?)
        ON CONFLICT(link) DO UPDATE SET title=excluded.title, created=excluded.created, updated=excluded.updated, author=excluded.author, avatar=excluded.avatar, rule=excluded.rule`,
		p.Title, p.Created, p.Updated, p.Link, p.Author, p.Avatar, p.Rule, nowOr(p.CreatedAt))
	if err != nil {
		return fmt.Errorf("upsert post %s: %w", p.Link, err)
	}
	return nil
}

// ListFriends 返回全部朋友，若 created_at 为空则在代码层兜底为当前时间。
func (s *SQLite) ListFriends(ctx context.Context) ([]model.Friend, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, link, avatar, COALESCE(error,''), created_at FROM friends ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query friends: %w", err)
	}
	defer rows.Close()
	var out []model.Friend
	for rows.Next() {
		var f model.Friend
		var createdAt sql.NullTime
		if err := rows.Scan(&f.Name, &f.Link, &f.Avatar, &f.Error, &createdAt); err != nil {
			return nil, fmt.Errorf("scan friends: %w", err)
		}
		if createdAt.Valid {
			f.CreatedAt = createdAt.Time
		} else {
			f.CreatedAt = time.Now()
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate friends: %w", err)
	}
	return out, nil
}

// ListPosts 返回全部文章，按 created 倒序。
func (s *SQLite) ListPosts(ctx context.Context) ([]model.Post, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT title, created, updated, link, author, avatar, rule, created_at FROM posts ORDER BY created DESC`)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()
	var out []model.Post
	for rows.Next() {
		var p model.Post
		var created sql.NullTime
		var updated sql.NullTime
		var createdAt sql.NullTime
		if err := rows.Scan(&p.Title, &created, &updated, &p.Link, &p.Author, &p.Avatar, &p.Rule, &createdAt); err != nil {
			return nil, fmt.Errorf("scan posts: %w", err)
		}
		if created.Valid {
			p.Created = created.Time
		}
		if updated.Valid {
			p.Updated = updated.Time
		}
		if createdAt.Valid {
			p.CreatedAt = createdAt.Time
		} else {
			p.CreatedAt = time.Now()
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return out, nil
}

// Stats 统计汇总：朋友总数/活跃数/异常数、文章总数、更新时间。
func (s *SQLite) Stats(ctx context.Context) (model.Stats, error) {
	var st model.Stats
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM friends`).Scan(&st.FriendsTotal); err != nil {
		return st, fmt.Errorf("count friends: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM friends WHERE (error IS NULL OR error = '')`).Scan(&st.FriendsAlive); err != nil {
		return st, fmt.Errorf("count friends alive: %w", err)
	}
	st.FriendsError = st.FriendsTotal - st.FriendsAlive
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM posts`).Scan(&st.PostsTotal); err != nil {
		return st, fmt.Errorf("count posts: %w", err)
	}
	st.UpdatedAt = time.Now()
	return st, nil
}

// CleanOldPosts 按天数阈值清理过期文章（基于 created 字段）。
func (s *SQLite) CleanOldPosts(ctx context.Context, days int) error {
	if days <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM posts WHERE created < datetime('now', ?)`, fmtDays(days))
	if err != nil {
		return fmt.Errorf("clean old posts: %w", err)
	}
	return nil
}

func fmtDays(days int) string { return fmt.Sprintf("-%d days", days) }
func nowOr(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
