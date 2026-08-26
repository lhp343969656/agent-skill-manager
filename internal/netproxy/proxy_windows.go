//go:build windows

package netproxy

import (
	"net/url"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// proxyFromWindowsSystem 从 Windows 系统代理设置（Internet Settings）读取代理。
// 读取 HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings 下的
// ProxyEnable（是否启用）和 ProxyServer（形如 "127.0.0.1:7890" 或 "http=host:port;https=host:port"）。
func proxyFromWindowsSystem() *url.URL {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enable != 1 {
		return nil
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil || strings.TrimSpace(server) == "" {
		return nil
	}
	server = strings.TrimSpace(server)

	// 形如 "https=host:port;http=host:port"，只取 https 段（GitHub 都是 https）
	if strings.Contains(server, "=") {
		for _, part := range strings.Split(server, ";") {
			if strings.HasPrefix(strings.ToLower(part), "https=") {
				addr := strings.TrimPrefix(part, "https=")
				addr = strings.TrimPrefix(addr, "HTTPS=")
				if addr != "" {
					return &url.URL{Scheme: "http", Host: addr}
				}
			}
		}
		return nil
	}

	// 形如 "127.0.0.1:7890"，直接作为 http 代理
	return &url.URL{Scheme: "http", Host: server}
}
