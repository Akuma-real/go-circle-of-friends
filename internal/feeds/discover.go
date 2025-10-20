// 包 feeds 负责订阅发现与解析：
// - DiscoverFeed：基于常见路径与 HTML <link> 自动发现订阅
// - ParseFeed：使用 gofeed 解析 RSS/Atom/JSON Feed 并归一化
package feeds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"go-circle-of-friends/internal/fetch"
	"go-circle-of-friends/internal/logx"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// DiscoverFeed 尝试常见端点与 HTML <link> 以发现订阅地址。
func DiscoverFeed(ctx context.Context, cl *fetch.Client, site string, feedSuffix string) (string, error) {
	// 若提供了 feedSuffix，则优先尝试
	candidates := []string{}
	if feedSuffix != "" {
		candidates = append(candidates,
			// 同时尝试根路径与基路径拼接两种语义
			joinURL(site, feedSuffix),
			joinURLDir(site, feedSuffix),
		)
	}
	// 先尝试以当前链接为目录基路径进行拼接（适配子路径站点）
	candidates = append(candidates,
		joinURLDir(site, "index.xml"),
		joinURLDir(site, "atom.xml"),
		joinURLDir(site, "rss.xml"),
		joinURLDir(site, "feed"),
		joinURLDir(site, "feed.xml"),
	)
	// 再尝试以站点根为基准的常见 endpoints
	candidates = append(candidates,
		// 常见通用 endpoints
		joinURL(site, "/feed"),
		joinURL(site, "/feed/"),
		joinURL(site, "/feed.xml"),
		joinURL(site, "/index.xml"),
		joinURL(site, "/atom.xml"),
		joinURL(site, "/rss.xml"),
		joinURL(site, "/rss2.xml"),
		joinURL(site, "/rss.php"),
		joinURL(site, "/feed.php"),
		joinURL(site, "/rss"),
		joinURL(site, "/atom"),
		joinURL(site, "/index.rss"),
		joinURL(site, "/index.atom"),
		// WordPress 兼容参数形式
		joinURL(site, "/?feed=rss2"),
		joinURL(site, "/?feed=atom"),
		joinURL(site, "/?feed=rss"),
		// 部分主题/平台的备用路径
		joinURL(site, "/feed/atom"),
		joinURL(site, "/posts/index.xml"),
		joinURL(site, "/blog/index.xml"),
		// JSON Feed（如存在）
		joinURL(site, "/index.json"),
		joinURL(site, "/feed.json"),
		// REST 风格
		joinURL(site, "/api/rss"),
	)

	// 全串行探测候选订阅，按顺序逐个尝试
	for _, u := range candidates {
		logx.Debugf("探测候选订阅：%s", u)
		if ok := probeFeed(ctx, cl, u); ok {
			return u, nil
		}
	}
	// 回退：抓取 HTML 并解析 <link> 标签
	resp, err := cl.Get(ctx, site)
	if err != nil {
		return "", fmt.Errorf("GET site %s: %w", site, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	var found string
	// 优先解析 rel=alternate 的订阅声明
	doc.Find("link").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		rel, _ := s.Attr("rel")
		t, _ := s.Attr("type")
		href, _ := s.Attr("href")
		lt := strings.ToLower(t)
		lr := strings.ToLower(rel)
		if strings.Contains(lr, "alternate") && (strings.Contains(lt, "rss") || strings.Contains(lt, "atom") || strings.Contains(lt, "json")) {
			found = joinURL(site, href)
			return false
		}
		// 若缺少 type，也尝试根据后缀判断（谨慎）
		if lt == "" {
			lh := strings.ToLower(href)
			if strings.HasSuffix(lh, ".xml") || strings.HasSuffix(lh, ".rss") || strings.HasSuffix(lh, ".atom") || strings.HasSuffix(lh, ".json") {
				found = joinURL(site, href)
				return false
			}
		}
		return true
	})
	if found != "" && probeFeed(ctx, cl, found) {
		logx.Debugf("从 <link> 发现订阅：%s", found)
		return found, nil
	}
	return "", fmt.Errorf("no feed discovered for %s", site)
}

// probeFeed 粗略探测 URL 是否为订阅（根据 Content-Type/状态码）。
func probeFeed(ctx context.Context, cl *fetch.Client, feedURL string) bool {
	// 为单个候选设置较短超时，避免串行探测拖慢整体速度
	prCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	resp, err := cl.Get(prCtx, feedURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// 读取少量内容用于嗅探，避免将 HTML 误判为订阅
	head, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "rss") || strings.Contains(ct, "atom") || strings.Contains(ct, "xml") {
		return true
	}
	if strings.Contains(ct, "json") || strings.Contains(ct, "feed+json") {
		lb := bytes.ToLower(head)
		if bytes.Contains(lb, []byte("jsonfeed")) || bytes.Contains(lb, []byte("\"version\":\"https://jsonfeed.org/version")) {
			return true
		}
		return false
	}
	// 按内容粗略嗅探 XML/JSON Feed 标记
	lb := strings.ToLower(string(head))
	if strings.Contains(lb, "<rss") || strings.Contains(lb, "<feed") || strings.Contains(lb, "<rdf") {
		return true
	}
	if strings.Contains(lb, "\"version\":\"https://jsonfeed.org/version") {
		return true
	}
	return false
}

// joinURL 将相对路径解析为绝对 URL。
func joinURL(base, ref string) string {
	if strings.HasPrefix(ref, "http") {
		return ref
	}
	u, err := url.Parse(base)
	if err != nil {
		return base + ref
	}
	ru, err := url.Parse(ref)
	if err != nil {
		return base + ref
	}
	return u.ResolveReference(ru).String()
}

// joinURLDir 将 base 视为目录进行相对拼接（即便 base 不以 / 结尾）。
// 常用于在子路径（如 https://host/blog）下尝试 blog/index.xml 等。
func joinURLDir(base, ref string) string {
	if strings.HasPrefix(ref, "http") {
		return ref
	}
	u, err := url.Parse(base)
	if err != nil {
		b := base
		if !strings.HasSuffix(b, "/") {
			b += "/"
		}
		return b + strings.TrimPrefix(ref, "/")
	}
	// 确保将 base 当作目录
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	ru, err := url.Parse(strings.TrimPrefix(ref, "/"))
	if err != nil {
		return u.String() + strings.TrimPrefix(ref, "/")
	}
	return u.ResolveReference(ru).String()
}

// ParseFeed 从订阅地址解析并返回归一化后的条目（最多返回 max 条，0 表示不限制）。
func ParseFeed(ctx context.Context, cl *fetch.Client, feedURL string, max int) ([]Item, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	p := gofeed.NewParser()
	// gofeed 不直接接收自定义 http.Client，因此先用自定义客户端抓取后再交给 gofeed 解析
	resp, err := cl.Get(reqCtx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("GET feed %s: %w", feedURL, err)
	}
	defer resp.Body.Close()
	feed, err := p.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse feed %s: %w", feedURL, err)
	}
	items := make([]Item, 0, len(feed.Items))
	for _, it := range feed.Items {
		item := Item{
			Title:   safe(it.Title),
			Link:    safe(it.Link),
			Author:  authorName(it),
			Updated: pickTime(it.UpdatedParsed, it.PublishedParsed),
			Created: pickTime(it.PublishedParsed, it.UpdatedParsed),
		}
		items = append(items, item)
		if max > 0 && len(items) >= max {
			break
		}
	}
	return items, nil
}

// Item 为解析后的文章临时结构（供上层转换为 model.Post）。
type Item struct {
	Title   string
	Link    string
	Author  string
	Created time.Time
	Updated time.Time
}

func pickTime(a, b *time.Time) time.Time {
	if a != nil {
		return *a
	}
	if b != nil {
		return *b
	}
	return time.Time{}
}

func authorName(it *gofeed.Item) string {
	if it.Author != nil {
		if it.Author.Name != "" {
			return it.Author.Name
		}
		if it.Author.Email != "" {
			return it.Author.Email
		}
	}
	return ""
}

func safe(s string) string { return strings.TrimSpace(s) }
