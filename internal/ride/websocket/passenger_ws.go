package websocket

import (
	"log"
	"net/http"
	"ride-hail/internal/common/auth"
	commonws "ride-hail/internal/common/websocket"
	"time"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func PassengerWSHandler(w http.ResponseWriter, r *http.Request, hub *commonws.Hub) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

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
		log.Printf("passenger WS user read error: %v", err)
		conn.Close()
		return
	}

	// Проверяем токен
	userID, err := auth.ValidateToken(authMsg.Token)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": "invalid token"})
		conn.Close()
		return
	}

	// Создаем клиента и регистрируем в Hub
	client := &commonws.Client{
		ID:   "passenger_" + userID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	log.Printf("🧍‍♀️ Passenger connected: %s", userID)

	// Периодически отправляем Ping
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("ping failed for passenger %s: %v", userID, err)
					conn.Close()
					return
				}
			}
		}
	}()

	// Читаем входящие сообщения (например, подтверждения или чаты)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("passenger %s disconnected: %v", userID, err)
			break
		}

		log.Printf("📨 Message from passenger %s: %s", userID, msg)
		// можно просто сохранить сообщение в канал (если нужно)
		hub.Broadcast <- msg
	}

	hub.Unregister <- client
	conn.Close()
	log.Printf("🚪 Passenger connection closed: %s", userID)
}
