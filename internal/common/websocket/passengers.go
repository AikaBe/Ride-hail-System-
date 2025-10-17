package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/model"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func PassengerWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "failed to upgrade", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	log.Println("Passenger connected")

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("auth read error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"auth_timeout"}`))
		return
	}

	var incoming model.Message
	_ = json.Unmarshal(msg, &incoming)

	if incoming.Type != "auth" {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid_auth_message"}`))
		return
	}

	if _, err := auth.ValidateToken(incoming.Token); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid_token"}`))
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(`{"status":"authenticated"}`))
	log.Println("Passenger authenticated")

	conn.SetPongHandler(func(appData string) error {
		log.Println("Pong received from passenger")
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Ping failed:", err)
				return
			}

		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}

			log.Printf("Passenger says: %s", msg)
			conn.WriteMessage(messageType, []byte("Server received: "+string(msg)))
		}
	}
}
