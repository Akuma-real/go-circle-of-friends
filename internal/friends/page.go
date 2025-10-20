// 包 friends 提供友链页解析：
// - 依据 rules.yaml 预设的 CSS 选择器获取 name/link/avatar
// - 支持 "选择器@属性" 以及 "||" 多方案回退与相对 URL 绝对化
package friends

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"go-circle-of-friends/internal/config"
	"go-circle-of-friends/internal/fetch"
	"go-circle-of-friends/internal/rules"
)

// ParseFriendsPage 根据选择器预设从友链页抽取朋友信息。
// 规则语法：
// - 文本：".name" 或 "."（取当前项文本）
// - 属性："a@href"/"img@src"/"@href"（当前项属性）
// - 回退：使用 "||" 连接多个候选，按先后尝试
func ParseFriendsPage(ctx context.Context, cl *fetch.Client, pageURL string, preset rules.Preset) ([]config.StaticFriend, error) {
	if preset.FriendsPage == nil {
		return nil, nil
	}
	resp, err := cl.Get(ctx, pageURL)
	if err != nil {
		return nil, fmt.Errorf("GET friends page %s: %w", pageURL, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		return nil, fmt.Errorf("parse friends page html: %w", err)
	}
	fp := preset.FriendsPage
	var out []config.StaticFriend
	doc.Find(fp.Item).Each(func(_ int, s *goquery.Selection) {
		name := getVal(s, fp.Name)
		link := abs(pageURL, getVal(s, fp.Link))
		avatar := abs(pageURL, getVal(s, fp.Avatar))
		name = strings.TrimSpace(name)
		if name == "" && link == "" {
			return
		}
		out = append(out, config.StaticFriend{
			Name:   name,
			Link:   link,
			Avatar: avatar,
		})
	})
	return out, nil
}

// getVal 解析表达式并支持使用 "||" 作为回退分隔，例如："a@href||@href" 或 ".name||.friend-name||."。
func getVal(scope *goquery.Selection, expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, p := range parts {
			if v := getValSingle(scope, strings.TrimSpace(p)); v != "" {
				return v
			}
		}
		return ""
	}
	return getValSingle(scope, expr)
}

// getValSingle 解析单个表达式：文本或 属性 读取。
func getValSingle(scope *goquery.Selection, expr string) string {
	if expr == "" {
		return ""
	}
	if expr == "." {
		return strings.TrimSpace(scope.Text())
	}
	if at := strings.Index(expr, "@"); at != -1 {
		sel := strings.TrimSpace(expr[:at])
		attr := strings.TrimSpace(expr[at+1:])
		if sel == "" {
			val, _ := scope.Attr(attr)
			return strings.TrimSpace(val)
		}
		if el := scope.Find(sel).First(); el != nil {
			val, _ := el.Attr(attr)
			return strings.TrimSpace(val)
		}
		return ""
	}
	if el := scope.Find(expr).First(); el != nil {
		return strings.TrimSpace(el.Text())
	}
	return ""
}

// abs 将相对链接转换为绝对 URL。
func abs(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	bu, err := url.Parse(base)
	if err != nil {
		return ref
	}
	ru, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return bu.ResolveReference(ru).String()
}
