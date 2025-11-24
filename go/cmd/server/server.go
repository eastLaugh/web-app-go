package main

import (
	"io/fs"
	"net/http"
	"strconv"

	"github.com/eastLaugh/web-app-go/go/internal/users"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Serve(fsys fs.FS) {

	// HTTP服务器
	// backend implements HttpBackendServer
	backend := gin.New()
	RegisterMiddleware(backend)
	RegisterRouter(backend, NewHttpBackendServerWithSqlite())

	// 文件服务器
	http.Handle("/api/", backend)
	http.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(fsys))))
	http.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))
	panic(http.ListenAndServe(":8080", nil))
}

func RegisterMiddleware(engine *gin.Engine) {
	engine.Use(gin.Recovery())
	engine.Use(gin.LoggerWithWriter(logrus.StandardLogger().Writer()))
}

func RegisterRouter(router gin.IRouter, backend HttpBackendServer) {
	//获取用户信息
	router.GET("/api/v1/users/:id", func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		user, err := backend.GetUser(ctx, id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		ctx.JSON(http.StatusOK, user)

	})

	//注册用户
	router.POST("/api/v1/users", func(ctx *gin.Context) {
		var user users.User
		if err := ctx.ShouldBindJSON(&user); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id, err := backend.CreateUser(ctx, user)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"id": id, "message": "user created"})
	})

	//恐慌
	router.GET("/api/v1/panic", func(ctx *gin.Context) {
		panic(nil)
	})
}
