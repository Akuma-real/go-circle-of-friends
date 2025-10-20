// 包 model 定义导出的数据模型（朋友/文章/统计/导出结构）。
package model

import "time"

// Friend 表示一个友链站点（聚合对象）。
type Friend struct {
	Name      string    `json:"name"`
	Link      string    `json:"link"`
	Avatar    string    `json:"avatar"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Post 为归一化后的文章条目。
type Post struct {
	Title     string    `json:"title"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	Link      string    `json:"link"`
	Author    string    `json:"author"`
	Avatar    string    `json:"avatar"`
	Rule      string    `json:"rule"`
	CreatedAt time.Time `json:"created_at"`
}

// Stats 为聚合统计信息。
type Stats struct {
	FriendsTotal int       `json:"friends_total"`
	FriendsAlive int       `json:"friends_alive"`
	FriendsError int       `json:"friends_error"`
	PostsTotal   int       `json:"posts_total"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Export 为极简导出的 data.json 顶层结构。
type Export struct {
	Stats   Stats    `json:"stats"`
	Friends []Friend `json:"friends"`
	Posts   []Post   `json:"posts"`
}
