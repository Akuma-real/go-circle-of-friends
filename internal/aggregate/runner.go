// 包 aggregate 负责主流程编排：
// - 合并静态与页面解析得到的朋友列表
// - 并发发现订阅并解析文章
// - 落库与过期清理
package aggregate

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-circle-of-friends/internal/config"
	"go-circle-of-friends/internal/feeds"
	"go-circle-of-friends/internal/fetch"
	"go-circle-of-friends/internal/friends"
	"go-circle-of-friends/internal/logx"
	"go-circle-of-friends/internal/model"
	"go-circle-of-friends/internal/rules"
	"go-circle-of-friends/internal/store"
)

// Runner 聚合执行器，持有配置/存储/HTTP 客户端/规则。
type Runner struct {
	cfg   *config.Config
	rules *rules.Rules
	fetch *fetch.Client
	store *store.SQLite
	// 简洁模式：仅收集内存数据，不落库
	buf *SimpleBuffer
}

// New 创建 Runner。
func New(cfg *config.Config, s *store.SQLite, cl *fetch.Client, rl *rules.Rules) *Runner {
	r := &Runner{cfg: cfg, store: s, fetch: cl, rules: rl}
	if cfg != nil && cfg.SimpleMode {
		r.buf = NewSimpleBuffer()
	}
	return r
}

// Run 执行一轮聚合：发现朋友→发现订阅→解析文章→清理过期。
func (r *Runner) Run(ctx context.Context) error {
	// 构建朋友列表（静态 + 页面来源）
	friendsList := dedup(r.cfg.StaticFriends)
	logx.Infof("静态朋友=%d，页面来源=%d", len(r.cfg.StaticFriends), len(r.cfg.LinkSources))
	for _, src := range r.cfg.LinkSources {
		if src.Type != "page" {
			continue
		}
		var preset rules.Preset
		if r.rules != nil {
			if p, ok := r.rules.GetPreset(src.Theme); ok {
				preset = p
			}
		}
		found, err := friends.ParseFriendsPage(ctx, r.fetch, src.URL, preset)
		if err != nil {
			logx.Warnf("解析友链页失败：%s 错误=%v", src.URL, err)
			continue
		}
		logx.Infof("%s 解析到 %d 位朋友", src.URL, len(found))
		friendsList = mergeDedup(friendsList, found)
	}
	if len(friendsList) == 0 {
		logx.Warnf("没有发现任何朋友（静态或页面）")
	}

	sem := make(chan struct{}, max(1, r.cfg.Concurrency.Fetch))
	var wg sync.WaitGroup
	for _, sf := range friendsList {
		sf := sf
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r.processFriend(ctx, sf)
		}()
	}
	wg.Wait()

	// 正常模式才清理数据库中过期文章；极简模式不使用数据库
	if r.buf == nil {
		if err := r.store.CleanOldPosts(ctx, r.cfg.OutdateCleanDays); err != nil {
			logx.Warnf("清理过期文章失败：%v", err)
		}
	}
	return nil
}

// processFriend 处理单个朋友：订阅发现→解析→写库。
func (r *Runner) processFriend(ctx context.Context, sf config.StaticFriend) {
	host := hostOf(sf.Link)
	f := model.Friend{
		Name:      sf.Name,
		Link:      sf.Link,
		Avatar:    sf.Avatar,
		CreatedAt: time.Now(),
	}
	// 发现订阅
	feedURL, err := feeds.DiscoverFeed(ctx, r.fetch, sf.Link, sf.FeedSuffix)
	if err != nil {
		f.Error = err.Error()
		if r.buf != nil {
			r.buf.AddFriend(f)
		} else {
			_ = r.store.UpsertFriend(ctx, f)
		}
		logx.Warnf("[%s|%s] 发现订阅失败：%v", sf.Name, host, err)
		return
	}
	f.Error = ""
	if r.buf != nil {
		r.buf.AddFriend(f)
	} else {
		if err := r.store.UpsertFriend(ctx, f); err != nil {
			logx.Warnf("写入朋友失败：%v", err)
		}
	}
	// 解析文章条目
	items, err := feeds.ParseFeed(ctx, r.fetch, feedURL, r.cfg.MaxPostsNum)
	if err != nil {
		logx.Warnf("[%s|%s] 解析订阅失败：%v", sf.Name, host, err)
		return
	}
	logx.Infof("[%s|%s] 文章解析完成：%d", sf.Name, host, len(items))
	for _, it := range items {
		p := model.Post{
			Title:     it.Title,
			Created:   it.Created,
			Updated:   it.Updated,
			Link:      it.Link,
			Author:    it.Author,
			Avatar:    sf.Avatar,
			Rule:      "feed",
			CreatedAt: time.Now(),
		}
		if r.buf != nil {
			r.buf.AddPost(p)
		} else {
			if err := r.store.UpsertPost(ctx, p); err != nil {
				logx.Warnf("写入文章失败：%v", err)
			}
		}
	}
}

// dedup 按 link 去重。
func dedup(in []config.StaticFriend) []config.StaticFriend {
	m := map[string]config.StaticFriend{}
	for _, f := range in {
		if f.Link == "" {
			continue
		}
		if _, ok := m[f.Link]; !ok {
			m[f.Link] = f
		}
	}
	out := make([]config.StaticFriend, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// mergeDedup 合并两个朋友切片并按 link 去重。
func mergeDedup(base []config.StaticFriend, add []config.StaticFriend) []config.StaticFriend {
	m := map[string]config.StaticFriend{}
	for _, f := range base {
		m[f.Link] = f
	}
	for _, f := range add {
		if f.Link == "" {
			continue
		}
		if _, ok := m[f.Link]; ok {
			continue
		}
		m[f.Link] = f
	}
	out := make([]config.StaticFriend, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// hostOf 提取链接的主机名，失败时做字符串兜底，便于日志定位。
func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	s := raw
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if j := strings.IndexAny(s, "/?#"); j >= 0 {
		s = s[:j]
	}
	return s
}

// BufferData 返回极简模式下收集的内存数据（朋友、文章）。
func (r *Runner) BufferData() ([]model.Friend, []model.Post) {
	if r == nil || r.buf == nil {
		return nil, nil
	}
	return r.buf.Snapshot()
}
