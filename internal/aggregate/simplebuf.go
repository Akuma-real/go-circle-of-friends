package aggregate

import (
	"sort"
	"sync"

	"go-circle-of-friends/internal/model"
)

// SimpleBuffer 在极简模式下收集聚合数据，避免落库。
type SimpleBuffer struct {
	mu      sync.Mutex
	friends map[string]model.Friend // key: link
	posts   map[string]model.Post   // key: link
}

func NewSimpleBuffer() *SimpleBuffer {
	return &SimpleBuffer{
		friends: make(map[string]model.Friend),
		posts:   make(map[string]model.Post),
	}
}

func (b *SimpleBuffer) AddFriend(f model.Friend) {
	if f.Link == "" {
		return
	}
	b.mu.Lock()
	b.friends[f.Link] = f
	b.mu.Unlock()
}

func (b *SimpleBuffer) AddPost(p model.Post) {
	if p.Link == "" {
		return
	}
	b.mu.Lock()
	b.posts[p.Link] = p
	b.mu.Unlock()
}

func (b *SimpleBuffer) AddPosts(list []model.Post) {
	b.mu.Lock()
	for _, p := range list {
		if p.Link == "" {
			continue
		}
		b.posts[p.Link] = p
	}
	b.mu.Unlock()
}

// Snapshot 返回副本：
// - friends 按名字排序
// - posts 按创建时间倒序
func (b *SimpleBuffer) Snapshot() ([]model.Friend, []model.Post) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fr := make([]model.Friend, 0, len(b.friends))
	for _, v := range b.friends {
		fr = append(fr, v)
	}
	sort.Slice(fr, func(i, j int) bool { return fr[i].Name < fr[j].Name })
	ps := make([]model.Post, 0, len(b.posts))
	for _, v := range b.posts {
		ps = append(ps, v)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Created.After(ps[j].Created) })
	return fr, ps
}
