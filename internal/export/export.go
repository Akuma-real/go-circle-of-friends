// 包 export 负责极简模式导出：将库中数据写为 data.json。
package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go-circle-of-friends/internal/model"
	"go-circle-of-friends/internal/store"
)

// ToJSON 查询统计/朋友/文章并写入 JSON 文件（带缩进格式）。
func ToJSON(ctx context.Context, s *store.SQLite, path string) error {
	friends, err := s.ListFriends(ctx)
	if err != nil {
		return fmt.Errorf("list friends: %w", err)
	}
	posts, err := s.ListPosts(ctx)
	if err != nil {
		return fmt.Errorf("list posts: %w", err)
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		return fmt.Errorf("stats: %w", err)
	}
	// 全局文章数上限保护：按时间倒序仅保留最新 150 篇）
	const maxExportPosts = 150
	if len(posts) > maxExportPosts {
		posts = posts[:maxExportPosts]
	}
	// 统计中的 posts_total 以导出数量为准，避免与上限不符
	stats.PostsTotal = len(posts)
	out := model.Export{Stats: stats, Friends: friends, Posts: posts}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode json to %s: %w", path, err)
	}
	return nil
}
