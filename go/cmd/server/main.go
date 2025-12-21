package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/eastLaugh/web-app-go/go/api"
	"github.com/eastLaugh/web-app-go/go/cmd/server/ports"
	"github.com/eastLaugh/web-app-go/go/pkg/tokens"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed dist/*
var dist embed.FS

func init() {
	_ = godotenv.Load()
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, nil)))
}

func main() {

	var fsys fs.FS
	fsys, _ = fs.Sub(dist, "dist")

	go Serve(fsys)
	// go ServeConsole(fsys)

	select {}
}

//go:embed template/*
var consoleTemplate embed.FS
var consoleTmpl = template.Must(template.ParseFS(consoleTemplate, "template/*"))

var serverChan chan *ports.Server = make(chan *ports.Server, 1)

func ServeConsole(fsys fs.FS) {
	mux := http.NewServeMux()
	server := <-serverChan
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		consoleTmpl.Execute(w, nil)
	})
	mux.HandleFunc("GET /rebuild-index", func(w http.ResponseWriter, r *http.Request) {
		if err := server.BuildRAGIndex(r.Context(), fsys); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "索引构建成功"})
	})

	panic(http.ListenAndServe("localhost:2333", mux))
}

func Serve(fsys fs.FS) {
	server, cancel := ports.NewServer(initDB(), initMongo())
	defer cancel()

	serverChan <- server

	mux := http.NewServeMux()

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		Middlewares: []api.MiddlewareFunc{tokens.Middleware},
	})))

	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("panic test")
	})

	// 文件服务器
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(fsys))))
	mux.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))

	handler := loggingMiddleware(recoveryMiddleware(mux))

	addr := os.Getenv("EASTLAUGH_ADDR")
	log.Printf("监听于 %s ...", addr)
	panic(http.ListenAndServe(addr, handler))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("请求", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
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
	// Sends a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		panic(err)
	}
	log.Printf("MongoDB 连接成功")
	return client
}

func initDB() *sql.DB {
	dsn, ok := os.LookupEnv("EASTLAUGH_DATABASE_DSN")
	if !ok {
		log.Printf("未设置 EASTLAUGH_DATABASE_DSN 环境变量")
		os.Exit(1)
	}

	driver, ok := os.LookupEnv("EASTLAUGH_DATABASE_DRIVER")
	if !ok {
		log.Printf("未设置 EASTLAUGH_DATABASE_DRIVER 环境变量")
		os.Exit(1)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		log.Printf("数据库 ping 失败: %v", err)
		os.Exit(1)
	}

	log.Printf("数据库连接成功")
	return db
}
