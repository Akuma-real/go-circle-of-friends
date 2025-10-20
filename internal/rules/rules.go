// 包 rules 负责加载并提供主题解析规则（rules.yaml），
// 以预设名（如 default/clarity）组织 CSS 选择器，用于友链页/文章页解析。
package rules

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rules 为抓取所需的选择器集合（朋友列表/文章页）。
// Rules 表示全部规则集合：键为预设名，值为具体规则。
type Rules struct {
	Presets map[string]Preset `yaml:",inline"`
}

// Preset 为单个主题预设的解析规则集合。
type Preset struct {
	FriendsPage *FriendsPage `yaml:"friends_page"`
}

// FriendsPage 描述友链页的选择器：
// - item：每个朋友条目容器
// - name/link/avatar：取文本或属性（支持 a@href / img@src）
type FriendsPage struct {
	Item   string `yaml:"item"`
	Name   string `yaml:"name"`
	Link   string `yaml:"link"`
	Avatar string `yaml:"avatar"`
}

// 备注：文章页解析规则已移除；当前通过订阅获取文章。

func Load(path string) (*Rules, error) {
	// 从文件加载 YAML 到 Rules.Presets
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open rules %s: %w", path, err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read rules %s: %w", path, err)
	}
	var r Rules
	if err := yaml.Unmarshal(b, &r.Presets); err != nil {
		return nil, fmt.Errorf("unmarshal rules %s: %w", path, err)
	}
	return &r, nil
}

// GetPreset 按名称获取预设（不区分大小写），若为空或不存在则回退到 "default"。
func (r *Rules) GetPreset(name string) (Preset, bool) {
	if r == nil || len(r.Presets) == 0 {
		return Preset{}, false
	}
	if name == "" {
		name = "default"
	}
	if p, ok := r.Presets[name]; ok {
		return p, true
	}
	// 不区分大小写匹配
	lower := strings.ToLower(name)
	for k, v := range r.Presets {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	if p, ok := r.Presets["default"]; ok {
		return p, true
	}
	for _, v := range r.Presets {
		return v, true
	}
	return Preset{}, false
}
