package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"time"

	"github.com/eastLaugh/web-app-go/go/pkg/tools"
)

func VectorSearch(ctx context.Context, args *struct{ Query string }) string {
	srv := ctx.Value(reflect.TypeFor[*Server]()).(*Server)
	s, err := srv.runVectorSearch(ctx, args.Query)
	if err != nil {
		return err.Error()
	}
	return s
}

func registerChatTools(_ *Server) *tools.Registry {
	return tools.New(
		VectorSearch,
		"在博客文档中搜索相关内容，返回最相似的文档片段。可用中文精简关键字。",
		Puzzle,
		"猜数字（1～100），报 Guess，返回猜大了/猜小了/猜中。assistant 和 user 均可以玩游戏。assistant 作为玩家时，需要多次调用此工具，并无需等待 user 回复。",
		Echo,
		"原样返回用户给的文本，用于测试",
		Now,
		"返回当前服务器时间，无参数",
		Add,
		"把两个整数相加，返回和",
		func(ctx context.Context, _ *struct{}) string {
			time.Sleep(time.Second * 10)
			return "睡眠10秒完成"
		},
		"睡眠10秒",
		RetrieveOnSale,
		"返回在售商品列表",
		CreatePaymentLink,
		"创建支付链接",
		CheckPaymentStatus,
		"检查支付状态",
	)
}

func Echo(ctx context.Context, args *struct{ Text string }) string {
	return args.Text
}

func Now(ctx context.Context, args *struct{}) string {
	return time.Now().Format(time.RFC3339)
}

func Add(ctx context.Context, args *struct {
	A int
	B int
}) string {
	return fmt.Sprintf("%d", args.A+args.B)
}

var answer = rand.N(100)

func Puzzle(ctx context.Context, args *struct {
	Guess int
}) string {
	if args.Guess > answer {
		return "猜大了"
	}
	if args.Guess < answer {
		return "猜小了"
	}
	return "真棒"
}

type Product struct {
	ID    string
	Name  string
	Price int
}

func RetrieveOnSale(ctx context.Context, _ *struct{}) string {
	var products []any

	products = append(products, Product{
		ID:    "1",
		Name:  "绫小路惠梨香的手办",
		Price: 2999,
	}, Product{
		ID:    "2",
		Name:  "天蓝色囊地鼠GOPHER玩偶-Z",
		Price: 200,
	}, Product{
		ID:    "3",
		Name:  "100%聚酯纤维无袖上衣男款2026夏季新装",
		Price: 199,
	})
	body, err := json.Marshal(products)
	if err != nil {
		return err.Error()
	}
	return string(body)
}

func CreatePaymentLink(ctx context.Context, args *struct {
	ID string
}) string {
	// return "https://pay.eastlaugh.com/pay?id=" + args.ID
	return "相关功能暂未实现"
}

func CheckPaymentStatus(ctx context.Context, args *struct {
	ID string
}) string {
	// return "用户已支付，即将送货上门"
	return "相关功能暂未实现"
}
