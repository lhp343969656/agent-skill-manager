// Package netproxy 提供统一的 HTTP Transport，使程序能跟随系统代理设置。
// 优先读取 Windows 系统代理（注册表），回退到环境变量代理。
package netproxy

import (
	"net/http"
	"net/url"
	"os"
	"runtime"
)

// NewTransport 返回一个配置好代理的 http.Transport。
// 在 Windows 上会读取系统代理设置（Internet Settings）；其他平台使用环境变量代理。
func NewTransport() *http.Transport {
	tr := &http.Transport{}
	proxy := systemProxy()
	if proxy != nil {
		tr.Proxy = http.ProxyURL(proxy)
	}
	return tr
}

// NewClient 返回一个使用系统代理的 http.Client。
func NewClient() *http.Client {
	return &http.Client{
		Transport: NewTransport(),
	}
}

// systemProxy 返回当前生效的代理 URL；无代理时返回 nil。
func systemProxy() *url.URL {
	// 1. 优先环境变量（HTTP_PROXY / HTTPS_PROXY / ALL_PROXY）
	if u := proxyFromEnv(); u != nil {
		return u
	}
	// 2. Windows：读取系统代理设置
	if runtime.GOOS == "windows" {
		if u := proxyFromWindowsSystem(); u != nil {
			return u
		}
	}
	return nil
}

// proxyFromEnv 从环境变量读取代理。
func proxyFromEnv() *url.URL {
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if v := os.Getenv(k); v != "" {
			if u, err := url.Parse(v); err == nil && u.Host != "" {
				return u
			}
		}
	}
	return nil
}
