package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"

	"github.com/eastLaugh/web-app-go/go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed dist/*
var dist embed.FS

func init() {
	gin.DefaultWriter = logrus.StandardLogger().Writer()
}

func main() {

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

	go Serve(fsys)

	util.OpenURL("http://localhost:8080")

	select {}
}
