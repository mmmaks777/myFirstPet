package user

import (
	"fmt"
	"net/http"

	t "pet/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func User(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPage := c.Param("user")

		loggedInUser, exist := c.Get("username")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "username not exist"})
			return
		}
		loggedInUser_id, exist := c.Get("user_id")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "user id not exist"})
			return
		}

		var posts []t.Post
		if err := DB.Where("author = ?", userPage).Find(&posts).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
			return
		}
		// fmt.Println(user_id)

		var creds t.Credentials
		if err := DB.Where("username = ?", userPage).First(&creds).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
			return
		}

		isFollowing := false
		var cnt int64
		if err := DB.Model(&t.Subscriptions{}).Where("user_id = ? and follower_id = ?", creds.ID, uint(loggedInUser_id.(float64))).Count(&cnt).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
			return
		}

		if cnt > 0 {
			isFollowing = true
		}

		var followers_cnt, following_cnt int64
		if err := DB.Model(&t.Subscriptions{}).Where("user_id = ?", creds.ID).Count(&followers_cnt).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
			return
		}
		if err := DB.Model(&t.Subscriptions{}).Where("follower_id = ?", creds.ID).Count(&following_cnt).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
			return
		}

		c.HTML(http.StatusOK, "user.html", gin.H{
			"User_id":        creds.ID,
			"LoggedInUser":   loggedInUser,
			"Username":       userPage,
			"Posts":          posts,
			"IsOwner":        loggedInUser == userPage,
			"IsFollowing":    isFollowing,
			"FollowersCount": followers_cnt,
			"FollowingCount": following_cnt,
		})
	}
}

func CreatePost(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exist := c.Get("username")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "username not exist"})
			return
		}

		post := t.Post{
			Title:   c.PostForm("title"),
			Content: c.PostForm("content"),
			Author:  user.(string),
		}

		post.Author = user.(string)

		if err := DB.Create(&post).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "could not create post"})
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/user/%s", user))
	}
}

func EditPost(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exist := c.Get("username")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "username not exist"})
			return
		}
		id := c.Param("id")
		post := t.Post{
			Title:   c.PostForm("title"),
			Content: c.PostForm("content"),
		}

		if err := DB.Model(&post).Where("id = ? AND author = ?", id, user).UpdateColumns(map[string]interface{}{
			"title":   post.Title,
			"content": post.Content,
		}).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "could not update post"})
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/user/%s", user))
	}
}

func DeletePosts(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		user, exist := c.Get("username")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "username not exist"})
			return
		}

		if id != "" {
			if err := DB.Delete(&t.Post{}, id).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "could not delete post"})
				return
			}
		} else {
			c.HTML(http.StatusBadRequest, "user.html", gin.H{"error": "invalid request"})
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/user/%s", user))
	}
}

func HandleFeed(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var posts []t.Post
		DB.Raw(`
			SELECT p.*
			FROM posts p
			JOIN subscriptions s ON p.author = (
				SELECT username FROM credentials WHERE id = s.user_id
			)
			WHERE s.follower_id = ?
			ORDER BY p.created_at DESC
		`, userID).Scan(&posts)

		c.HTML(http.StatusOK, "feed.html", gin.H{
			"posts": posts,
		})
	}
}
