package tokens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var tokenPwd = os.Getenv("EASTLAUGH_TOKEN_PWD")

type Payload struct {
	Email  string `json:"email"`
	Expire int64  `json:"expire"`
}

func (t Payload) Export() (string, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)

	hash := hmac.New(sha256.New, []byte(tokenPwd))
	hash.Write(payload)

	return encoded + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil)), nil
}

func (t *Payload) Import(token string) error {
	if t == nil {
		return errors.New("t 为 nil")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return fmt.Errorf("无效的 token: %s", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("无效的 Payload: %s", parts[0])
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("解码签名失败: %s", parts[1])
	}

	hash := hmac.New(sha256.New, []byte(tokenPwd))
	hash.Write(payload)

	if !hmac.Equal(signature, hash.Sum(nil)) {
		return fmt.Errorf("无效的签名: %s", parts[1])
	}

	return json.Unmarshal(payload, t)
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		if i := strings.Index(x, ","); i >= 0 {
			x = strings.TrimSpace(x[:i])
		} else {
			x = strings.TrimSpace(x)
		}
		if x != "" {
			return x
		}
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

// RequireAuth 从 ctx 取 email，非登录用户（guest 或类型异常）直接 panic。
func RequireAuth(ctx context.Context) string {
	email := ctx.Value("email").(string)
	if strings.HasPrefix(email, "guest:") {
		panic("tokens: guest not allowed")
	}
	return email
}

// Middleware 只写 context：有有效 token 设 email，无/无效设 "guest:"+IP，一定 next。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		email := "guest:" + clientIP(r)
		token := r.Header.Get("Authorization")
		if token != "" {
			token = strings.TrimPrefix(token, "Bearer ")
			if token != "" {
				var payload Payload
				if payload.Import(token) == nil && payload.Expire >= time.Now().Unix() {
					email = payload.Email
				}
			}
		}
		ctx = context.WithValue(ctx, "email", email)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
