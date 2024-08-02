package messenger

import (
	"net/http"
	"pet/types"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID       uint
	Username string
	Socket   *websocket.Conn
	Send     chan []byte
	Partner  *Client
}

type ClientManager struct {
	Clients    map[uint]*Client
	Register   chan *Client
	Unregister chan *Client
	sync.RWMutex
}

var manager = ClientManager{
	Clients:    make(map[uint]*Client),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
}

func StartManager() {
	go manager.start()
}

func HandleChats(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		var chats []types.Chat
		DB.Raw(`
			SELECT u.username AS username, (
				SELECT content
				FROM messages
				WHERE (sender_id = u.id AND receiver_id = ?) OR (sender_id = ? AND receiver_id = u.id)
				ORDER BY created_at DESC
				LIMIT 1
			) AS last_message
			FROM credentials u
			WHERE u.id IN (
				SELECT DISTINCT CASE
					WHEN m.sender_id = ? THEN m.receiver_id
					WHEN m.receiver_id = ? THEN m.sender_id
				END
				FROM messages m
				WHERE m.sender_id = ? OR m.receiver_id = ?
			)
		`, userID, userID, userID, userID, userID, userID).Scan(&chats)

		c.HTML(http.StatusOK, "chats.html", gin.H{
			"userID":   userID,
			"username": username,
			"chats":    chats,
		})
	}
}

func HandleChat(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		partner := c.Param("partner")

		var creds types.Credentials
		if err := DB.Where("username = ?", partner).First(&creds).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "messenger.html", gin.H{"error": "username not exist"})
			return
		}

		var messages []types.Message
		if err := DB.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", userID, creds.ID, creds.ID, userID).Find(&messages).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "messenger.html", gin.H{"error": "cannot load messages"})
			return
		}

		c.HTML(http.StatusOK, "messenger.html", gin.H{
			"userID":   userID,
			"username": username,
			"partner":  partner,
			"messages": messages,
		})
	}
}

func HandleWebSocket(DB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, _ := c.Get("user_id")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			http.NotFound(c.Writer, c.Request)
			return
		}
		user := c.Query("user")
		partner := c.Query("partner") // ID партнера для чата

		var creds types.Credentials
		if err := DB.Where("username = ?", partner).First(&creds).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "messenger.html", gin.H{"error": "username not exist"})
			return
		}

		client := &Client{
			ID:       uint(userId.(float64)),
			Username: user,
			Socket:   conn,
			Send:     make(chan []byte),
		}

		manager.Register <- client

		manager.Lock()
		if partner, ok := manager.Clients[creds.ID]; ok {
			client.Partner = partner
			partner.Partner = client
		}
		manager.Unlock()

		go client.read(DB, creds.ID)
		go client.write()
	}
}

func (manager *ClientManager) start() {
	for {
		select {
		case conn := <-manager.Register:
			manager.Lock()
			manager.Clients[conn.ID] = conn
			manager.Unlock()
		case conn := <-manager.Unregister:
			manager.Lock()
			if partner := conn.Partner; partner != nil {
				partner.Partner = nil
			}
			if _, ok := manager.Clients[conn.ID]; ok {
				close(conn.Send)
				delete(manager.Clients, conn.ID)
			}
			manager.Unlock()
		}
	}
}

func (c *Client) read(DB *gorm.DB, partnerId uint) {
	defer func() {
		manager.Unregister <- c
		c.Socket.Close()
	}()

	for {
		_, message, err := c.Socket.ReadMessage()
		if err != nil {
			manager.Unregister <- c
			c.Socket.Close()
			break
		}

		messageDB := types.Message{
			Sender_id:   c.ID,
			Receiver_id: partnerId,
			Content:     string(message),
		}

		if err := DB.Create(&messageDB).Error; err != nil {
			manager.Unregister <- c
			c.Socket.Close()
			break
		}

		if c.Partner != nil {
			c.Partner.Send <- message
		}
	}
}

func (c *Client) write() {
	defer c.Socket.Close()
	for message := range c.Send {
		if err := c.Socket.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
	c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
}
