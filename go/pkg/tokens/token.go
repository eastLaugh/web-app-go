package tokens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eastLaugh/web-app-go/go/internal/api"
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

// Middleware 返回一个标准库风格的中间件
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否需要认证
		scopes := r.Context().Value(api.BearerAuthScopes)
		if scopes == nil {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "未授权", http.StatusUnauthorized)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			http.Error(w, "未授权", http.StatusUnauthorized)
			return
		}
		var payload Payload
		err := payload.Import(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if payload.Expire < time.Now().Unix() {
			http.Error(w, "token 已过期", http.StatusUnauthorized)
			return
		}

		// 直接替换上下文
		ctx := r.Context()
		ctx = context.WithValue(ctx, "email", payload.Email)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
