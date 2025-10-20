// 包 config 负责加载与校验应用配置（settings.yaml），
// 对外提供结构体 Config 及默认值/合法性校验。
package config

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// 仅保留当前需要的字段，避免过度设计（KISS/YAGNI）。
type Config struct {
	LinkSources      []LinkSource   `yaml:"LINK"`
	StaticFriends    []StaticFriend `yaml:"SETTINGS_FRIENDS_LINKS"`
	MaxPostsNum      int            `yaml:"MAX_POSTS_NUM"`
	OutdateCleanDays int            `yaml:"OUTDATE_CLEAN"`
	SimpleMode       bool           `yaml:"SIMPLE_MODE"`
	ResetOnStart     bool           `yaml:"RESET_ON_START"`
	Database         Database       `yaml:"DATABASE"`
	Concurrency      Concurrency    `yaml:"CONCURRENCY"`
	Proxy            Proxy          `yaml:"PROXY"`
	LogLevel         string         `yaml:"LOG_LEVEL"`
	LogFormat        string         `yaml:"LOG_FORMAT"` // text|json|pretty
	LogLocale        string         `yaml:"LOG_LOCALE"` // zh-CN|en
	LogColor         string         `yaml:"LOG_COLOR"`  // auto|always|never
}

type LinkSource struct {
	// Type：来源类型，当前支持 page（友链页按选择器解析）
	Type  string `yaml:"type"` // page|json (page rules are WIP; we support json later)
	URL   string `yaml:"url"`
	Theme string `yaml:"theme"`
}

type StaticFriend struct {
	// FeedSuffix：可选订阅后缀（如 /atom.xml /feed），用于提升发现命中率
	Name       string `yaml:"name"`
	Link       string `yaml:"link"`
	Avatar     string `yaml:"avatar"`
	FeedSuffix string `yaml:"feed_suffix"`
}

type Database struct {
	Type string `yaml:"type"` // sqlite (default)
	DSN  string `yaml:"dsn"`  // ./data.db
}

type Concurrency struct {
	Fetch int `yaml:"fetch"`
	Retry int `yaml:"retry"`
}

type Proxy struct {
	HTTP  string `yaml:"http"`
	HTTPS string `yaml:"https"`
}

func Load(path string) (*Config, error) {
	// Load 从文件读取 YAML 并反序列化为 Config，同时进行基础校验与默认值填充。
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %s: %w", path, err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("unmarshal config %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &c, nil
}

func (c *Config) Validate() error {
	// Validate 负责合法性检查与默认值设置，避免在业务层分散判空逻辑。
	if c.MaxPostsNum < 0 {
		return errors.New("MAX_POSTS_NUM must be >= 0")
	}
	if c.OutdateCleanDays < 0 {
		return errors.New("OUTDATE_CLEAN must be >= 0")
	}
	if c.Database.Type == "" {
		c.Database.Type = "sqlite"
	}
	if c.Database.Type != "sqlite" {
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}
	if c.Database.DSN == "" {
		c.Database.DSN = "./data.db"
	}
	if c.Concurrency.Fetch <= 0 {
		c.Concurrency.Fetch = 8
	}
	if c.Concurrency.Retry < 0 {
		c.Concurrency.Retry = 2
	}
	if c.LogFormat == "" {
		c.LogFormat = "pretty"
	}
	if c.LogLocale == "" {
		c.LogLocale = "zh-CN"
	}
	if c.LogColor == "" {
		c.LogColor = "auto"
	}
	// ResetOnStart 默认为 false，显式开启时才执行清理
	return nil
}
