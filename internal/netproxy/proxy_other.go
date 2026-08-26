//go:build !windows

package netproxy

import "net/url"

// proxyFromWindowsSystem 在非 Windows 平台始终返回 nil（由环境变量代理兜底）。
func proxyFromWindowsSystem() *url.URL {
	return nil
}
