package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-skill-manager/internal/netproxy"
)

// oauthClientID 是 GitHub OAuth 应用 Client ID。
// 开发者需在 GitHub -> Settings -> Developer settings -> OAuth Apps 注册应用获取。
// 这是一个占位值，实际使用时请替换为真实 Client ID。
const oauthClientID = "Ov23liIHnhwGurqBUM0j"

// authHTTPClient 使用跟随系统代理的 HTTP 客户端，确保设备流授权能访问 GitHub。
var authHTTPClient = &http.Client{Transport: netproxy.NewTransport()}

// DeviceAuth 描述一次设备流授权的持久化状态。
type DeviceAuth struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	VerifyURI  string `json:"verification_uri"`
	Interval   int    `json:"interval"`
	ExpiresIn  int    `json:"expires_in"`
}

// StartDeviceAuth 发起 GitHub OAuth 设备流授权，返回供用户打开的验证信息。
// 用户需在浏览器打开 VerificationURI 输入 UserCode 以完成授权。
func StartDeviceAuth(ctx context.Context) (*DeviceAuth, error) {
	form := url.Values{}
	form.Set("client_id", oauthClientID)
	form.Set("scope", "repo")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := authHTTPClient.Do(req)
	if err != nil {
		// 网络不通（连接失败/超时）时给出明确提示，避免用户误以为是程序没反应
		return nil, fmt.Errorf("无法连接 GitHub 授权服务，请检查网络或代理后重试（%v）", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("发起授权失败: GitHub 返回错误 HTTP %d，请稍后重试", resp.StatusCode)
	}

	var d DeviceAuth
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if d.DeviceCode == "" || d.UserCode == "" {
		return nil, fmt.Errorf("授权信息不完整")
	}
	if d.VerifyURI == "" {
		d.VerifyURI = "https://github.com/login/device"
	}
	if d.Interval <= 0 {
		d.Interval = 5
	}
	return &d, nil
}

// PollDeviceAuthPoll 轮询一段设备流授权结果，返回 access_token。
// 轮询期间用户需在浏览器完成授权。deviceCode 来自 StartDeviceAuth。
func PollDeviceAuthPoll(ctx context.Context, deviceCode string, interval int) (string, error) {
	if interval <= 0 {
		interval = 5
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		form := url.Values{}
		form.Set("client_id", oauthClientID)
		form.Set("device_code", deviceCode)
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := authHTTPClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("轮询授权结果失败: %w", err)
		}

		var body struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		decErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if decErr != nil {
			resp.Body.Close()
			return "", decErr
		}

		if body.AccessToken != "" {
			return body.AccessToken, nil
		}
		// authorization_pending：用户尚未完成授权，继续轮询
		// slow_down：请求过快，增加间隔并继续
		if body.Error == "slow_down" {
			interval += 5
		}
	}
}
