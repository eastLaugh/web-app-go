package tokens

import (
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

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/gin-gonic/gin"
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

func Middleware(ctx *gin.Context) {
	// logrus.Debugf("Middleware: %v", ctx.Request.URL.Path)
	_, ok := ctx.Get(api.BearerAuthScopes)
	if !ok {
		ctx.Next()
		return
	}

	token := ctx.GetHeader("Authorization")
	if token == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	var payload Payload
	err := payload.Import(token)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if payload.Expire < time.Now().Unix() {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token 已过期"})
		return
	}
	ctx.Set("email", payload.Email)
	ctx.Next()
}
