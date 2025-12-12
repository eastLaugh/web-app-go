package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/eastLaugh/web-app-go/go/cmd/server/ports"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

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
	// gin.DefaultWriter = logrus.StandardLogger().Writer()
	// gin.DefaultErrorWriter = logrus.StandardLogger().Writer()

	err := godotenv.Load()
	if err != nil {
		logrus.Infof("未加载本地 .env 文件: %v", err)
	}

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

	//util.OpenURL("http://localhost:8080")

	select {}
}

func Serve(fsys fs.FS) {

	// HTTP服务器
	v1 := gin.New()
	v1.Use(gin.Recovery())
	v1.Use(gin.LoggerWithWriter(logrus.StandardLogger().Writer()))
	v1.GET("/panic", func(c *gin.Context) {
		panic("panic test")
	})

	server, cancel := ports.NewServer(initDB(), initMongo())
	defer cancel()

	api.RegisterHandlersWithOptions(v1, server, api.GinServerOptions{
		BaseURL:     "/api/v1/",
		Middlewares: []api.MiddlewareFunc{tokens.Middleware},
	})

	http.Handle("/api/v1/", v1)

	// 文件服务器
	http.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(fsys))))
	http.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))

	port := os.Getenv("EASTLAUGH_PORT")
	logrus.Infof("监听于 %s", port)
	panic(http.ListenAndServe(port, nil))
}

func initMongo() *mongo.Client {
	var uri string
	if uri = os.Getenv("EASTLAUGH_MONGODB_URI"); uri == "" {
		log.Fatal("未设置 EASTLAUGH_MONGODB_URI 环境变量")
	}
	// Uses the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	// Defines the options for the MongoDB client
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	// Creates a new client and connects to the server
	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// Sends a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result); err != nil {
		panic(err)
	}
	logrus.Info("MongoDB 连接成功")
	return client
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
