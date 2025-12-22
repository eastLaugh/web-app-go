package main

import (
	"context"
	"embed"
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
	go ServeConsole(fsys)

	select {}
}

//go:embed template/*
var consoleTemplate embed.FS
var consoleTmpl = template.Must(template.ParseFS(consoleTemplate, "template/*"))

var serverChan chan *ports.Server = make(chan *ports.Server, 1)

func Serve(fsys fs.FS) {
	server, cancel := ports.NewServer(initMongo())
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
