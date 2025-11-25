package main

import (
	"database/sql"
	"embed"
	"flag"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/eastLaugh/web-app-go/go/cmd/server/ports"
	"github.com/eastLaugh/web-app-go/go/internal/util"
	"github.com/eastLaugh/web-app-go/go/internal/util/tokens"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed dist/*
var dist embed.FS

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.DateTime,
		ForceColors:     true,
	})
	gin.DefaultWriter = logrus.StandardLogger().Writer()
	gin.DefaultErrorWriter = logrus.StandardLogger().Writer()
}

func main() {

	local := flag.Bool("local", false, "使用本地文件系统而不是嵌入的文件系统")
	flag.Parse()

	var fsys fs.FS
	if *local {
		fsys = os.DirFS("dist")
	} else {
		fsys, _ = fs.Sub(dist, "dist")
	}
	logrus.Debugf("使用文件系统: %T", fsys)

	go Serve(fsys)

	util.OpenURL("http://localhost:8080")

	select {}
}

func Serve(fsys fs.FS) {
	// 数据库连接
	db := initDB()
	defer db.Close()

	// HTTP服务器
	v1 := gin.New()
	v1.Use(gin.Recovery())
	v1.Use(gin.LoggerWithWriter(logrus.StandardLogger().Writer()))
	v1.GET("/panic", func(c *gin.Context) {
		panic("panic test")
	})

	server := ports.NewServer(db)
	defer server.Close()
	api.RegisterHandlersWithOptions(v1, server, api.GinServerOptions{
		BaseURL:     "/api/v1/",
		Middlewares: []api.MiddlewareFunc{tokens.Middleware},
	})

	http.Handle("/api/v1/", v1)

	// 文件服务器
	http.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(fsys))))
	http.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))
	panic(http.ListenAndServe(":8080", nil))
}

func initDB() *sql.DB {
	dsn, ok := os.LookupEnv("EASTLAUGH_DATABASE_DSN")
	if !ok {
		logrus.Fatal("未设置 EASTLAUGH_DATABASE_DSN 环境变量")
	}

	driver, ok := os.LookupEnv("EASTLAUGH_DATABASE_DRIVER")
	if !ok {
		logrus.Fatal("未设置 EASTLAUGH_DATABASE_DRIVER 环境变量")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		logrus.Fatalf("数据库连接失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		logrus.Fatalf("数据库 ping 失败: %v", err)
	}

	logrus.Info("数据库连接成功")
	return db
}
