package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/eastLaugh/web-app-go/go/pkg/tools"
)

func vector_search(ctx context.Context, args *struct{ Query string }) string {
	meta := ctx.Value(ConvMetaKey{}).(*ConvMeta)
	s, err := meta.Server.runVectorSearch(ctx, args.Query)
	if err != nil {
		return err.Error()
	}
	return s
}

func set_conversation_title(ctx context.Context, args *struct{ Title string }) string {
	meta := ctx.Value(ConvMetaKey{}).(*ConvMeta)
	if err := meta.Server.userRepo.SetConversationTitle(ctx, meta.Email, meta.ConversationID, args.Title); err != nil {
		return err.Error()
	}
	return "已设置对话标题"
}

func registerChatTools(_ *Server) *tools.Registry {
	return tools.New(
		set_conversation_title,
		"设置当前对话的标题，用于在历史列表中展示。标题应简短，建议不超过20字。",
		vector_search,
		"在博客文档中搜索相关内容，返回最相似的文档片段。可用中文精简关键字。",
		puzzle,
		"猜数字（1～100），报 Guess，返回猜大了/猜小了/猜中。assistant 和 user 均可以玩游戏。assistant 作为玩家时，需要多次调用此工具，并无需等待 user 回复。",
		echo,
		"原样返回用户给的文本，用于测试",
		now,
		"返回当前服务器时间，无参数",
		add,
		"把两个整数相加，返回和",
		func(ctx context.Context, _ *struct{}) string {
			time.Sleep(time.Second * 10)
			return "睡眠10秒完成"
		},
		"睡眠10秒",
		retrieve_on_sale,
		"返回在售商品列表",
		create_payment_link,
		"创建支付链接",
		check_payment_status,
		"检查支付状态",
		tools.Fetch_web_page,
		"",
		Send_email,
		"SMTP 发送邮件",
	)
}

func echo(ctx context.Context, args *struct{ Text string }) string {
	return args.Text
}

func now(ctx context.Context, args *struct{}) string {
	return time.Now().Format(time.RFC3339)
}

func add(ctx context.Context, args *struct {
	A int
	B int
}) string {
	return fmt.Sprintf("%d", args.A+args.B)
}

var answer = rand.N(100)

func puzzle(ctx context.Context, args *struct {
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

func retrieve_on_sale(ctx context.Context, _ *struct{}) string {
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

func create_payment_link(ctx context.Context, args *struct {
	ID string
}) string {
	// return "https://pay.eastlaugh.com/pay?id=" + args.ID
	return "相关功能暂未实现"
}

func check_payment_status(ctx context.Context, args *struct {
	ID string
}) string {
	// return "用户已支付，即将送货上门"
	return "相关功能暂未实现"
}

func Send_email(ctx context.Context, args *struct {
	To      string `description:"收件人邮箱地址"`
	Subject string `description:"邮件主题"`
	Body    string `description:"邮件内容"`
}) string {

	meta := ctx.Value(ConvMetaKey{}).(*ConvMeta)
	if args.To != meta.Email && args.To != "east_laugh@qq.com" {
		return "无法发送邮件：只允许发送给用户或 east_laugh@qq.com\n如果用户为访客，无法发送给用户，请引导用户通过邮箱登录"
	}

	err := tools.SendMail(args.Subject, args.Body, args.To)
	if err != nil {
		return err.Error()
	}
	return "邮件发送成功"
}
