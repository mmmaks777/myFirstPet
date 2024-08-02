package follow

import (
	"fmt"
	"net/http"
	t "pet/types"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Follow(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		follower_id, exist := c.Get("user_id")
		if !exist {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": "user id not exist"})
			return
		}

		user_idStr := c.Param("id")
		user_id, err := strconv.Atoi(user_idStr)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
		}

		newSubscription := t.Subscriptions{
			UserID:     uint(user_id),
			FollowerID: uint(follower_id.(float64)),
		}

		var subscription t.Subscriptions
		result := DB.Unscoped().Model(&t.Subscriptions{}).Where("user_id = ? and follower_id = ?", newSubscription.UserID, newSubscription.FollowerID).First(&subscription)

		if result.Error == nil {
			if subscription.DeletedAt.Valid {
				if err := DB.Unscoped().Model(&subscription).UpdateColumn("deleted_at", nil).Error; err != nil {
					c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
					return
				}
			} else {
				if err := DB.Delete(&t.Subscriptions{}, subscription).Error; err != nil {
					c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
					return
				}
			}
		} else {
			if result.Error == gorm.ErrRecordNotFound {
				if err := DB.Create(&newSubscription).Error; err != nil {
					c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error()})
					return
				}
			} else {
				c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": result.Error})
				return
			}
		}

		var username string
		if err := DB.Model(&t.Credentials{}).Where("id = ?", user_id).Select("username").First(&username).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "user.html", gin.H{"error": err.Error})
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/user/%s", username))
	}
}

func Followers_handler(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user_id := c.Param("id")
		var username string
		if err := DB.Model(t.Credentials{}).Where("id = ?", user_id).Select("username").First(&username).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "followers.html", gin.H{"error": "username error"})
			return
		}

		var followers []string
		if err := DB.Table("credentials").Select("credentials.username").
			Joins("join subscriptions on subscriptions.follower_id = credentials.id").
			Where("subscriptions.user_id = ?", user_id).Scan(&followers).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "followers.html", gin.H{"error": "followers error"})
			return
		}

		c.HTML(http.StatusOK, "followers.html", gin.H{
			"Username":  username,
			"Followers": followers,
		})
	}
}

func Following_handler(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user_id := c.Param("id")
		var username string
		if err := DB.Model(t.Credentials{}).Where("id = ?", user_id).Select("username").First(&username).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "following.html", gin.H{"error": "username error"})
			return
		}

		var following []string
		if err := DB.Table("credentials").Select("credentials.username").
			Joins("join subscriptions on subscriptions.user_id = credentials.id").
			Where("subscriptions.follower_id = ?", user_id).Scan(&following).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "following.html", gin.H{"error": "followers error"})
			return
		}

		c.HTML(http.StatusOK, "following.html", gin.H{
			"Username":  username,
			"Following": following,
		})
	}
}
