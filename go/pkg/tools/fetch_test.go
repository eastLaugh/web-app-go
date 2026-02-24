package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetch_web_page_emptyURL(t *testing.T) {
	ctx := context.Background()
	out := Fetch_web_page(ctx, &struct {
		URL string `description:"要抓取的网页地址，需带 http(s) 协议"`
	}{})
	if out != "URL 不能为空" {
		t.Errorf("empty URL: got %q", out)
	}
}

func TestFetch_web_page_ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Title</title></head><body><p>Hello</p><script>ignore</script><p>World</p></body></html>`))
	}))
	defer server.Close()

	ctx := context.Background()
	out := Fetch_web_page(ctx, &struct {
		URL string `description:"要抓取的网页地址，需带 http(s) 协议"`
	}{URL: server.URL})
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "World") {
		t.Errorf("expected Hello and World in text, got: %s", out)
	}
	if strings.Contains(out, "<p>") || strings.Contains(out, "ignore") {
		t.Errorf("expected no tags or script content, got: %s", out)
	}
}
