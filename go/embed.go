package lib

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
)

//go:embed .env
var env string

func LoadEnv(writer io.Writer) {
	scanner := bufio.NewScanner(strings.NewReader(env))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		os.Setenv(parts[0], parts[1])
		fmt.Fprintf(writer, "[ %s := %s ]\n", parts[0], os.Getenv(parts[0]))
	}
}
