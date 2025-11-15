package server

import (
	"database/sql"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/eastLaugh/web-app-go/go/iface"
	"github.com/eastLaugh/web-app-go/go/users"
	"github.com/gin-gonic/gin"
)

func Serve(fsys fs.FS) {

	//backend implements http.Handler
	backend := gin.New()
	Registerar(backend, iface.NewBackendServer(users.SQLiteUserRepo{
		Db: func() *sql.DB {
			db, err := sql.Open("sqlite3", "data/user.db")
			if err != nil {
				panic(err)
			}
			// 确保表存在且email字段有UNIQUE约束
			db.Exec(`
				CREATE TABLE IF NOT EXISTS users (
					id INTEGER PRIMARY KEY AUTOINCREMENT, 
					email TEXT UNIQUE, 
					name TEXT
				)
			`)
			return db
		}(),
	}))
	http.Handle("/api/", backend)

	handler := http.FileServer(http.FS(fsys))
	http.Handle("/app/", http.StripPrefix("/app/", handler))
	http.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func Registerar(router gin.IRouter, backend iface.IBackend) {
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
}
