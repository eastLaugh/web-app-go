package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"

	"github.com/eastLaugh/web-app-go/go/server"
	"github.com/eastLaugh/web-app-go/go/util"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed dist/*
var dist embed.FS

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	local := flag.Bool("local", false, "使用本地文件系统而不是嵌入的文件系统")
	flag.Parse()

	var fsys fs.FS
	if *local {
		fsys = os.DirFS("dist")
		log.Println("本地")
	} else {
		fsys, _ = fs.Sub(dist, "dist")
		log.Println("嵌入")
	}

	go server.Serve(fsys)

	util.OpenURL("http://localhost:8080")

	select {}
}
