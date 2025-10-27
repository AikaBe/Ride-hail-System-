package websocket

import (
	"log"
	"net/http"
	"ride-hail/internal/user/jwt"
	"time"

	commonws "ride-hail/internal/common/websocket"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func DriverWSHandler(w http.ResponseWriter, r *http.Request, hub *commonws.Hub, jwtManager *jwt.Manager) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		conn.Close()
	}()
	done := make(chan struct{})
	// Устанавливаем таймауты и обработчик pong
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Читаем сообщение авторизации
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&authMsg); err != nil {
		log.Printf("Driver WS read auth error: %v", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "auth failed"))
		return
	}

	// Проверяем токен
	claims, err := jwtManager.ValidateToken(authMsg.Token)
	if err != nil {
		log.Printf("invalid token for passenger: %v", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "invalid token"))
		return
	}

	// Создаем клиента и регистрируем в Hub
	client := &commonws.Client{
		ID:   "driver_" + claims.UserID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	log.Printf("🚪 Driver connection closed: %s", client.ID)

	// Периодически отправляем Ping
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				if err != nil {
					log.Printf("ping failed for driver %s: %v", claims.UserID, err)
					return
				}
			}
		}
	}()

	go func() {
		for msg := range client.Send {
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Ошибка отправки драйверу %s: %v", client.ID, err)
				break
			}
		}
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("driver %s disconnected unexpectedly: %v", claims.UserID, err)
			} else {
				log.Printf("driver %s disconnected", claims.UserID)
			}
			break
		}

		log.Printf("📨 Message from driver %s: %s", claims.UserID, msg)
		hub.Broadcast <- msg
	}

	go hub.ListenDriverMessages(client)
	go hub.UpdateLocationWS(client)

	<-done
	hub.Unregister <- client
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	log.Printf("🚪 Passenger connection closed: %s", claims.UserID)
}
