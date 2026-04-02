package wechat

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client interface {
	CodeToSession(ctx context.Context, code string) (string, error)
}

type HTTPClient struct {
	appID     string
	appSecret string
	mock      bool
	client    *http.Client
}

type codeToSessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func New(appID, appSecret string, mock bool) *HTTPClient {
	return &HTTPClient{
		appID:     appID,
		appSecret: appSecret,
		mock:      mock,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *HTTPClient) CodeToSession(ctx context.Context, code string) (string, error) {
	if c.mock || c.appID == "" || c.appSecret == "" {
		sum := sha1.Sum([]byte(code))
		return "mock_" + hex.EncodeToString(sum[:8]), nil
	}

	values := url.Values{}
	values.Set("appid", c.appID)
	values.Set("secret", c.appSecret)
	values.Set("js_code", code)
	values.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.weixin.qq.com/sns/jscode2session?"+values.Encode(), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result codeToSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat login failed: %s", result.ErrMsg)
	}
	if result.OpenID == "" {
		return "", fmt.Errorf("wechat login returned empty openid")
	}
	return result.OpenID, nil
}
