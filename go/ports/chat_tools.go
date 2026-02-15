package ports

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/eastLaugh/web-app-go/go/pkg/tools"
)

func VectorSearch(ctx context.Context, args *struct{ Query string }) (string, error) {
	srv := ctx.Value(Server{}).(*Server)
	return srv.runVectorSearch(ctx, args.Query)
}

func registerChatTools(_ *Server) *tools.Registry {
	return tools.New(
		VectorSearch,
		"在博客文档中搜索相关内容，返回最相似的文档片段。可用中文精简关键字。",
		Puzzle,
		"猜一个100以内的数字。答案由服务端生成，user 和 assistant 均不知道答案，且均可作为玩家进行游玩。",
		Echo,
		"原样返回用户给的文本，用于测试",
		Now,
		"返回当前服务器时间，无参数",
		Add,
		"把两个整数相加，返回和",
	)
}

func Echo(ctx context.Context, args *struct{ Text string }) (string, error) {
	return args.Text, nil
}

func Now(ctx context.Context, args *struct{}) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}

func Add(ctx context.Context, args *struct {
	A int
	B int
}) (string, error) {
	return fmt.Sprintf("%d", args.A+args.B), nil
}

var answer = rand.N(100)

func Puzzle(ctx context.Context, args *struct {
	Guess int
}) (string, error) {
	if args.Guess > answer {
		return "", errors.New("猜大了")
	}
	if args.Guess < answer {
		return "", errors.New("猜小了")
	}
	return "真棒", nil
}
