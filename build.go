// go run build.go

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// 构建前端
	fmt.Println("构建前端项目...")
	cmd1 := exec.Command("npm", "run", "build")
	cmd1.Dir = filepath.Join(wd, "solid-project")
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	if err := cmd1.Run(); err != nil {
		log.Fatalf("前端构建失败: %v", err)
	}
	fmt.Println("前端构建完成")

	// 构建 Go 程序
	fmt.Println("构建 Go 程序...")
	cmd2 := exec.Command("go", "build", "-o", "../main", ".")
	cmd2.Dir = filepath.Join(wd, "go")
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		log.Fatalf("Go 构建失败: %v", err)
	}
	fmt.Println("Go 构建完成")

	// 等待用户按键
	fmt.Println("\n按 Enter 键启动 ./main...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadBytes('\n')

	// 启动 main 程序
	fmt.Println("启动 ./main...")
	cmd3 := exec.Command("./main")
	cmd3.Dir = wd
	cmd3.Stdout = os.Stdout
	cmd3.Stderr = os.Stderr
	if err := cmd3.Run(); err != nil {
		log.Fatalf("程序运行失败: %v", err)
	}
}
