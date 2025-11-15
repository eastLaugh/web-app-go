package token_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/eastLaugh/web-app-go/go/util/token"
)

func TestOne(t *testing.T) {
	mytoken := token.One{
		Email:  "test@test.com",
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}
	tokenStr, err := mytoken.Export()
	if err != nil {
		t.Fatal(err)
	}
	println(tokenStr)

	//解码
	var res token.One
	err = res.Import(tokenStr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%#v", res)
}
