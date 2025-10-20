// 包 logx 是对标准库 slog 的薄封装：
// - 支持级别/格式/语言/颜色配置
// - 提供 pretty 中文输出（[调试]/[信息]/[警告]/[错误]）
// - 通过 Debugf/Infof/Warnf/Errorf 暴露，便于将来替换底层实现（DIP）
package logx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// Init 根据 level/format/locale/colorMode 初始化全局日志器。
// 采用 slog 默认 Handler（json/text）或内置 PrettyHandler（中文美化）。
func Init(level, format, locale, colorMode string) {
	lv := parseSlogLevel(level)
	opts := &slog.HandlerOptions{Level: lv, AddSource: false}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "pretty", "":
		handler = NewPrettyHandler(os.Stdout, lv, locale, colorMode)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// parseSlogLevel 将字符串级别解析为 slog.Leveler。
func parseSlogLevel(s string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "none", "silent", "off":
		var l slog.Level = 100 // silence all
		return l
	case "info", "":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

// 便捷函数：格式化并按级别输出
func Debugf(format string, v ...any) { slog.Debug(fmt.Sprintf(format, v...)) }
func Infof(format string, v ...any)  { slog.Info(fmt.Sprintf(format, v...)) }
func Warnf(format string, v ...any)  { slog.Warn(fmt.Sprintf(format, v...)) }
func Errorf(format string, v ...any) { slog.Error(fmt.Sprintf(format, v...)) }

// PrettyHandler：最小可用的中文美化输出（可选彩色），仅用于人读；支持中英文标签。
type PrettyHandler struct {
	w      io.Writer
	level  slog.Leveler
	locale string
	color  bool
	mu     *sync.Mutex
	attrs  []slog.Attr
	group  string
}

// NewPrettyHandler 创建中文美化 Handler。
func NewPrettyHandler(w io.Writer, lv slog.Leveler, locale string, colorMode string) slog.Handler {
	if w == nil {
		w = os.Stdout
	}
	if locale == "" {
		locale = "zh-CN"
	}
	ph := &PrettyHandler{w: w, level: lv, locale: locale, mu: &sync.Mutex{}}
	ph.color = shouldColor(w, colorMode)
	return ph
}

// Enabled 根据配置的最低级别判定是否输出。
func (h *PrettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	// 与已配置的最低等级比较
	if ll, ok := h.level.(slog.Level); ok {
		return l >= ll && ll < 100
	}
	return true
}

// Handle 格式化输出：时间 + 等级 + 消息 + 扁平化属性
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var buf bytes.Buffer
	// 时间
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	buf.WriteString(ts.Format("2006-01-02 15:04:05"))
	buf.WriteString(" ")
	// 等级
	lvl := levelLabel(h.locale, r.Level)
	if h.color {
		lvl = colorize(lvl, r.Level)
	}
	buf.WriteString(lvl)
	buf.WriteString(" ")
	// 消息
	buf.WriteString(r.Message)
	// 附加属性（展平成 k=v）
	attrs := make([]slog.Attr, 0, len(h.attrs))
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	if len(attrs) > 0 {
		buf.WriteString(" ")
		for i, a := range attrs {
			if i > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(a.Key)
			buf.WriteString("=")
			buf.WriteString(a.Value.String())
		}
	}
	buf.WriteByte('\n')
	_, err := h.w.Write(buf.Bytes())
	return err
}

// WithAttrs 附加属性（本项目未大量使用）。
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(cp.attrs, attrs...)
	return &cp
}

// WithGroup 属性分组（本项目未大量使用）。
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	cp := *h
	if cp.group == "" {
		cp.group = name
	} else {
		cp.group += "." + name
	}
	return &cp
}

// levelLabel 根据语言返回等级标签。
func levelLabel(locale string, l slog.Level) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		switch l {
		case slog.LevelDebug:
			return "[调试]"
		case slog.LevelInfo:
			return "[信息]"
		case slog.LevelWarn:
			return "[警告]"
		case slog.LevelError:
			return "[错误]"
		default:
			return fmt.Sprintf("[L%d]", l)
		}
	}
	switch l {
	case slog.LevelDebug:
		return "[DEBUG]"
	case slog.LevelInfo:
		return "[INFO]"
	case slog.LevelWarn:
		return "[WARN]"
	case slog.LevelError:
		return "[ERROR]"
	default:
		return fmt.Sprintf("[L%d]", l)
	}
}

// shouldColor 判断是否启用颜色：遵循 LOG_COLOR 与 NO_COLOR。
func shouldColor(w io.Writer, mode string) bool {
	// 遵循 NO_COLOR 环境变量
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "always":
		return true
	case "never":
		return false
	case "auto", "":
		// 简单的 TTY 检测：仅在字符设备上启用彩色输出
		if f, ok := w.(*os.File); ok {
			if fi, err := f.Stat(); err == nil {
				return (fi.Mode() & os.ModeCharDevice) != 0
			}
		}
		return false
	default:
		return false
	}
}

// colorize 按等级包裹 ANSI 颜色码。
func colorize(s string, l slog.Level) string {
	// ANSI 颜色码
	code := ""
	switch l {
	case slog.LevelDebug:
		code = "90" // bright black
	case slog.LevelInfo:
		code = "36" // cyan
	case slog.LevelWarn:
		code = "33" // yellow
	case slog.LevelError:
		code = "31" // red
	default:
		code = "0"
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
