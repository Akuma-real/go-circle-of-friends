// 包 fetch 封装 HTTP 客户端（代理/超时/重试），用于抓取网页与订阅。
package fetch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client 为带重试的 HTTP 客户端。
type Client struct {
	http  *http.Client
	retry int
}

// Options 为客户端构造参数。
type Options struct {
	ProxyHTTP  string
	ProxyHTTPS string
	Timeout    time.Duration
	Retry      int
}

// New 创建客户端，支持 http/https 代理与基础超时配置。
func New(opts Options) (*Client, error) {
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			if req.URL.Scheme == "https" && opts.ProxyHTTPS != "" {
				return url.Parse(opts.ProxyHTTPS)
			}
			if req.URL.Scheme == "http" && opts.ProxyHTTP != "" {
				return url.Parse(opts.ProxyHTTP)
			}
			return http.ProxyFromEnvironment(req)
		},
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	cl := &http.Client{Transport: transport}
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}
	cl.Timeout = opts.Timeout
	return &Client{http: cl, retry: opts.Retry}, nil
}

// Get 请求带有指数退避（简单线性回退）重试。
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	attempts := c.retry + 1
	for i := 0; i < attempts; i++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			lastErr = fmt.Errorf("new request: %w", reqErr)
			break
		}
		// 使用常见浏览器 UA，减少 403/反爬误判；支持环境变量覆盖（COF_UA）
		ua := os.Getenv("COF_UA")
		if ua == "" {
			ua = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
		}
		req.Header.Set("User-Agent", ua)
		resp, err := c.http.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		if err == nil {
			lastErr = fmt.Errorf("http status: %s", resp.Status)
			if resp.Body != nil {
				resp.Body.Close()
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(i+1) * 300 * time.Millisecond):
		}
	}
	return nil, lastErr
}

// 备注：若某些站点仍返回 403，可按需设置环境变量 COF_UA 覆盖 UA。
