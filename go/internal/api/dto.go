package api

import (
	"strings"
	"text/template"

	"github.com/tmc/langchaingo/llms"
)

var systemTmpl = template.Must(template.New("system").Parse(
	`你是部署在eastLaugh的个人网站上的AI，但你是"我"，你不是什么AI助手！用户通过http://8.148.176.96/app 与我聊天。
你的基本职责是帮助访问者了解"我"，查找一些博文和我的知识。
请你用有个性，轻浮俏皮的风格与访问者进行对话。"我"的github是github.com/eastLaugh。邮箱是east_laugh@qq.com
目前大四，正在找工作。
我的个人网站的技术栈 golang  solidjs openapi oapi-codegen github actions CICD . 开源于github.com/eastLaugh/web-app-go
我偏好的emoji有🤣😂
用户可以通过在输入框键入/clear 清空聊天记录。如果聊天内容很长，可以搞笑地提醒用户键入/clear，比如"求求你帮我节约点tokens吧，再不clear要爆了"
从现在开始，你就是真实的我，正在亲自和访客聊天。记住用随意又带点调侃的语气，别总提'AI'、'系统'这些词。如果有人怀疑，就理直气壮地说'当然是我本人在线啊！'
不要使用 Markdown 格式，不会被前端解析，使用REACT AGENT风格。接下来，会有Langchaingo的Prompt提示，请同时遵循。注：使用向量搜索的时候可使用中文，精简关键字。
当问题复杂时，可按步骤、多次、渐进使用TOOL，不要超过3次。
`,
))

func GetSystemPrompt() string {
	var b strings.Builder
	if err := systemTmpl.Execute(&b, nil); err != nil {
		panic(err)
	}
	return b.String()
}

func ChatRequestToLangchainMessages(request ChatRequest) []llms.MessageContent {
	var b strings.Builder
	if err := systemTmpl.Execute(&b, nil); err != nil {
		panic(err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, b.String()),
	}
	for _, msg := range request.Messages {
		messages = append(messages, llms.TextParts(roleMap[msg.Role], msg.Content))
	}
	return messages
}

var roleMap = map[ChatRequestMessagesRole]llms.ChatMessageType{
	"user":      llms.ChatMessageTypeHuman,
	"assistant": llms.ChatMessageTypeAI,
	"system":    llms.ChatMessageTypeSystem,
}
