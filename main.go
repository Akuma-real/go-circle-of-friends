// 命令行入口：
// - 解析 flags 与 settings.yaml/rules.yaml
// - 初始化日志、HTTP 客户端、数据库
// - 支持友链页发现调试（-discover）与极简导出（data.json）
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"go-circle-of-friends/internal/aggregate"
	"go-circle-of-friends/internal/config"
	"go-circle-of-friends/internal/export"
	"go-circle-of-friends/internal/fetch"
	"go-circle-of-friends/internal/friends"
	"go-circle-of-friends/internal/logx"
	"go-circle-of-friends/internal/rules"
	"go-circle-of-friends/internal/store"
)

func main() {
	var (
		configPath = flag.String("config", "settings.yaml", "path to settings.yaml")
		rulesPath  = flag.String("rules", "rules.yaml", "path to rules.yaml (optional)")
		exportPath = flag.String("export", "data.json", "export json path when SIMPLE_MODE=true")
		discover   = flag.Bool("discover", false, "print discovered friends from LINK page sources and exit")
	)
	flag.Parse()

	// 1) 加载配置与规则
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	var rl *rules.Rules
	if *rulesPath != "" {
		if r, err := rules.Load(*rulesPath); err == nil {
			rl = r
		} else {
			log.Printf("load rules failed: %v", err)
		}
	}
	// 2) 初始化日志：级别/格式/语言/颜色
	logx.Init(cfg.LogLevel, cfg.LogFormat, cfg.LogLocale, cfg.LogColor)

	// 3) 初始化 HTTP 客户端（含代理与重试）
	cl, err := fetch.New(fetch.Options{
		ProxyHTTP:  cfg.Proxy.HTTP,
		ProxyHTTPS: cfg.Proxy.HTTPS,
		Timeout:    25 * time.Second,
		Retry:      cfg.Concurrency.Retry,
	})
	if err != nil {
		log.Fatalf("http client: %v", err)
	}

	ctx := context.Background()
	if *discover {
		// 4) 调试：仅解析友链页并打印结果后退出
		total := 0
		for _, src := range cfg.LinkSources {
			if src.Type != "page" {
				continue
			}
			var preset rules.Preset
			if rl != nil {
				if p, ok := rl.GetPreset(src.Theme); ok {
					preset = p
				}
			}
			list, err := friends.ParseFriendsPage(ctx, cl, src.URL, preset)
			if err != nil {
				logx.Errorf("解析友链页失败：%s 错误=%v", src.URL, err)
				continue
			}
			logx.Infof("%s 解析到 %d 位朋友", src.URL, len(list))
			for _, f := range list {
				logx.Infof("- 名称=%q 链接=%s 头像=%s", f.Name, f.Link, f.Avatar)
			}
			total += len(list)
		}
		if total == 0 {
			logx.Warnf("未从页面来源发现朋友，请检查 LINK.url 与 rules.yaml 选择器。")
		}
		return
	}

	// 5) 数据存储：极简模式不打开数据库；正常模式打开并按需重置
	var st *store.SQLite
	if !cfg.SimpleMode {
		var err error
		st, err = store.OpenSQLite(cfg.Database.DSN)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		defer st.Close()
		if cfg.ResetOnStart {
			if err := st.Reset(ctx); err != nil {
				logx.Warnf("启动清理数据库失败：%v", err)
			} else {
				logx.Infof("已清理数据库表（friends/posts）")
			}
		}
	} else if cfg.ResetOnStart {
		// 极简模式仅删除导出文件，不操作数据库
		logx.Infof("极简模式：跳过数据库打开与清理")
	}
	if cfg.ResetOnStart && *exportPath != "" {
		if err := os.Remove(*exportPath); err == nil {
			logx.Infof("已删除导出文件：%s", *exportPath)
		}
	}

	// 6) 运行聚合流程
	run := aggregate.New(cfg, st, cl, rl)
	logx.Infof("开始聚合：极简模式=%v", cfg.SimpleMode)
	if err := run.Run(ctx); err != nil {
		logx.Errorf("运行失败：%v", err)
		os.Exit(1)
	}

	if cfg.SimpleMode {
		// 7) 极简导出：只导出 JSON，跳过写库
		fr, ps := run.BufferData()
		if err := export.ToJSONData(ctx, fr, ps, *exportPath); err != nil {
			log.Fatalf("export json: %v", err)
		}
		logx.Infof("已导出 %s", *exportPath)
	}
}
