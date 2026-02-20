package prompt

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed system.txt
var systemTmplBytes string

var systemTmpl = template.Must(template.New("_").Parse(systemTmplBytes))

func GetSystemPrompt(email string) string {
	var b strings.Builder
	if err := systemTmpl.Execute(&b, map[string]any{"Email": email}); err != nil {
		panic(err)
	}
	return b.String()
}
