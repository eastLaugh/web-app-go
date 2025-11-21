package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var tokenPwd = os.Getenv("EASTLAUGH_TOKEN_PWD")

type One struct {
	Email  string `json:"email"`
	Expire int64  `json:"expire"`
}

func (t One) Export() (string, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)

	hash := hmac.New(sha256.New, []byte(tokenPwd))
	hash.Write(payload)

	return encoded + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil)), nil
}

func (t *One) Import(token string) error {
	if t == nil {
		return errors.New("t is nil")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid token : %s", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid Payload : %s", parts[0])
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode Signature : %s", parts[1])
	}

	hash := hmac.New(sha256.New, []byte(tokenPwd))
	hash.Write(payload)

	if !hmac.Equal(signature, hash.Sum(nil)) {
		return fmt.Errorf("invalid Signature : %s", parts[1])
	}

	return json.Unmarshal(payload, t)
}
