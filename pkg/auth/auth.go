package auth

import (
	"fmt"
	"net/http"
	"time"

	t "pet/types"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtKey = []byte("secret_key")

type Auth struct {
	DB *gorm.DB
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Register(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		creds := t.Credentials{
			Username: c.Request.FormValue("username"),
			Password: c.Request.FormValue("password"),
		}

		if err := DB.Where("username = ?", creds.Username).First(&t.Credentials{}).Error; err == nil {
			c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "User with this username already exist"})
			return
		}

		hashedPassword, err := HashPassword(creds.Password)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "error of hashing password"})
			return
		}

		creds.Password = hashedPassword

		if err := DB.Create(&creds).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "error creating user"})
			return
		}
		// log.Println(creds)

		c.Redirect(http.StatusFound, "/login")
	}
}

func Login(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		creds := t.Credentials{
			Username: c.Request.FormValue("username"),
			Password: c.Request.FormValue("password"),
		}

		var storedCreds t.Credentials

		if err := DB.Where("username = ?", creds.Username).First(&storedCreds).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect login or password"})
		}

		if !CheckPasswordHash(creds.Password, storedCreds.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect login or password"})
		}

		expirationTime := time.Now().Add(72 * time.Hour)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"User_id":  storedCreds.ID,
			"Username": creds.Username,
			"Exp":      expirationTime,
		})
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create token"})
			return
		}

		c.SetCookie("token", tokenString, int(expirationTime.Unix()), "/", "localhost", false, true)

		// c.HTML(http.StatusOK, "login.html", nil)
		c.Redirect(http.StatusFound, fmt.Sprintf("/user/%s", creds.Username))
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			// c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected singing method: %v", t.Header["alg"])
			}
			return jwtKey, nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"Error": err.Error(),
			})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("username", claims["Username"])
			c.Set("user_id", claims["User_id"])
			// fmt.Println(claims["Username"])
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Next()
	}
}
