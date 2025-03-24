package main

import (
	"log"
	"net/http"
	"text/template"
	"time"

	"pet/internal/follow"
	"pet/internal/messenger"
	"pet/internal/user"
	"pet/pkg/auth"
	"pet/pkg/db"

	"github.com/gin-gonic/gin"
)

func formatDate(t time.Time) string {
	return t.Format("02 Jan 2006")
}

func main() {
	DB := db.Connect()

	r := gin.Default()

	r.SetFuncMap(template.FuncMap{
		"formatDate": formatDate,
	})

	r.LoadHTMLGlob("../../web/templates/*")
	r.Static("/css", "../../web/static/css")

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	r.POST("/login", auth.Login(DB))
	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})
	r.POST("/register", auth.Register(DB))
	r.GET("/user/:user", auth.AuthMiddleware(), user.User(DB))
	r.POST("/user/addpost", auth.AuthMiddleware(), user.CreatePost(DB))
	r.POST("/user/delpost/:id", auth.AuthMiddleware(), user.DeletePosts(DB))
	r.POST("/user/editpost/:id", auth.AuthMiddleware(), user.EditPost(DB))
	r.POST("/follow/:id", auth.AuthMiddleware(), follow.Follow(DB))
	r.GET("/followers/:id", auth.AuthMiddleware(), follow.Followers_handler(DB))
	r.GET("/following/:id", auth.AuthMiddleware(), follow.Following_handler(DB))
	r.GET("/feed", auth.AuthMiddleware(), user.HandleFeed(DB))

	r.GET("/chats", auth.AuthMiddleware(), messenger.HandleChats(DB))
	r.GET("/messenger/:partner", auth.AuthMiddleware(), messenger.HandleChat(DB))
	r.GET("/ws", auth.AuthMiddleware(), messenger.HandleWebSocket(DB))

	messenger.StartManager()

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server run failed!!!: ", err)
	}
}
