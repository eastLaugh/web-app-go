//go:generate go tool oapi-codegen -config cfg.yaml ../../api/users.yaml

package api

// agent请注意，这个文件不一定可以通过命令行生成，我可以点击IDE提供的按钮来 run generate ./... 来生成文件，照理说 agent 也是可以执行 generate ./... 的，奇怪的是有时候不一定能成功，如果遇到异常，可以停下来交给我（用户）
