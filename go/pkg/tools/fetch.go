package tools

import (
	"context"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const maxBody = 512 * 1024

func Fetch_web_page(ctx context.Context, args *struct {
	URL string `description:"要抓取的网页地址，需带 http(s) 协议"`
}) string {
	if args.URL == "" {
		return "URL 不能为空"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.URL, nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; fetch_web_page/1)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "HTTP " + resp.Status
	}
	body := io.LimitReader(resp.Body, maxBody)
	b, err := io.ReadAll(body)
	if err != nil {
		return err.Error()
	}
	text := htmlToText(b)
	if utf8.RuneCountInString(text) > 8000 {
		runes := []rune(text)
		text = string(runes[:8000]) + "\n…(已截断)"
	}
	return text
}

func htmlToText(raw []byte) string {
	var out strings.Builder
	tkn := html.NewTokenizer(strings.NewReader(string(raw)))
	for {
		tt := tkn.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			s := strings.TrimSpace(html.UnescapeString(string(tkn.Text())))
			if s != "" {
				if out.Len() > 0 {
					out.WriteByte(' ')
				}
				out.WriteString(s)
			}
		}
		if tt == html.StartTagToken {
			name, _ := tkn.TagName()
			switch string(name) {
			case "script", "style":
				for tkn.Next() != html.EndTagToken {
				}
			case "p", "br", "div", "li", "tr":
				if out.Len() > 0 {
					out.WriteString("\n")
				}
			}
		}
	}
	return strings.TrimSpace(out.String())
}
