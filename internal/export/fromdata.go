package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go-circle-of-friends/internal/model"
)

// ToJSONData 直接将内存中的 friends/posts 写成 data.json，带全局上限与统计。
func ToJSONData(ctx context.Context, friends []model.Friend, posts []model.Post, path string) error {
	// 全局文章数上限保护，与 ToJSON 保持一致
	const maxExportPosts = 150
	if len(posts) > maxExportPosts {
		posts = posts[:maxExportPosts]
	}
	// 统计
	alive := 0
	for _, f := range friends {
		if f.Error == "" {
			alive++
		}
	}
	st := model.Stats{
		FriendsTotal: len(friends),
		FriendsAlive: alive,
		FriendsError: len(friends) - alive,
		PostsTotal:   len(posts),
		UpdatedAt:    time.Now(),
	}
	out := model.Export{Stats: st, Friends: friends, Posts: posts}
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
