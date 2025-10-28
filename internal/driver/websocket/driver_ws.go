package websocket

import (
	"context"
	"log"
	"net/http"
	"ride-hail/internal/driver/service"
	"ride-hail/internal/user/jwt"
	"time"

	commonws "ride-hail/internal/common/websocket"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func DriverWSHandler(w http.ResponseWriter, r *http.Request, hub *commonws.Hub, jwtManager *jwt.Manager, svc *service.DriverService) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	// Читаем сообщение авторизации
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&authMsg); err != nil {
		log.Printf("❌ Driver WS read auth error: %v", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "auth failed"))
		conn.Close()
		return
	}

	// Проверяем токен
	claims, err := jwtManager.ValidateToken(authMsg.Token)
	if err != nil {
		log.Printf("❌ Invalid token for driver: %v", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "invalid token"))
		conn.Close()
		return
	}

	// Проверяем, что пользователь действительно водитель
	if claims.Role != "DRIVER" {
		log.Printf("❌ User is not a driver: %s", claims.UserID)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "user is not a driver"))
		conn.Close()
		return
	}

	// Создаем клиента и регистрируем в Hub
	client := &commonws.Client{
		ID:   "driver_" + claims.UserID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	hub.Register <- client

	log.Printf("🚗 Driver connected: %s", client.ID)

	// Устанавливаем таймауты
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Горутина для отправки сообщений водителю
	go func() {
		defer func() {
			hub.Unregister <- client
			conn.Close()
			log.Printf("🚪 Driver connection closed: %s", client.ID)
		}()

		for {
			select {
			case message, ok := <-client.Send:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("❌ Error sending to driver %s: %v", client.ID, err)
					return
				}
			}
		}
	}()

	// Горутина для ping-pong
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("❌ Ping failed for driver %s: %v", client.ID, err)
					return
				}
			}
		}
	}()

	// Запускаем обработку сообщений от водителя
	go hub.ListenDriverMessages(client)

	// Запускаем обработку ответов для MQ
	go svc.SendToMq(context.Background())
	go svc.UpdateLocationWS(context.Background())

	// Ждем закрытия соединения
	<-make(chan struct{})
}
