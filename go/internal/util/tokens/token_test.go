package tokens_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/eastLaugh/web-app-go/go/internal/util/tokens"
	"github.com/google/uuid"
)

func TestPayload(t *testing.T) {
	oldPayload := tokens.Payload{
		Email:  uuid.New().String(),
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}

	token, err := oldPayload.Export()
	if err != nil {
		t.Fatal(err)
	}

	//验证
	var newPayload tokens.Payload
	err = newPayload.Import(token)
	if err != nil {
		t.Fatal(err)
	}

	if newPayload != oldPayload {
		t.Fatal("token not equal")
	}

}

func TestPayload_Export(t *testing.T) {
	os.Setenv("EASTLAUGH_TOKEN_PWD", "12312dsaa")
	payload := tokens.Payload{
		Email:  uuid.New().String(),
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}
	token, err := payload.Export()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(token)
}
