package websocket

import (
	"encoding/json"
	"net/http"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/model"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func PassengerWSHandler(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws_upgrade_failed", "Failed to upgrade WebSocket", requestID, "", err.Error(), "")
		http.Error(w, "failed to upgrade", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	logger.Info("ws_passenger_connected", "Passenger connected", requestID, "")

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, msg, err := conn.ReadMessage()
	if err != nil {
		logger.Error("ws_auth_read_failed", "Failed to read auth message", requestID, "", err.Error(), "")
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"auth_timeout"}`))
		return
	}

	var incoming model.Message
	_ = json.Unmarshal(msg, &incoming)

	if incoming.Type != "auth" {
		logger.Warn("ws_invalid_auth_message", "Invalid auth message type", requestID, "", "")
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid_auth_message"}`))
		return
	}

	if _, err := auth.ValidateToken(incoming.Token); err != nil {
		logger.Warn("ws_invalid_token", "Passenger token invalid", requestID, "", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid_token"}`))
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(`{"status":"authenticated"}`))
	logger.Info("ws_passenger_authenticated", "Passenger successfully authenticated", requestID, "")

	conn.SetPongHandler(func(appData string) error {
		logger.Debug("ws_pong_received", "Pong received from passenger", requestID, "")
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("ws_ping_failed", "Ping to passenger failed", requestID, "", err.Error(), "")
				return
			}

		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Error("ws_read_failed", "Failed to read passenger message", requestID, "", err.Error(), "")
				return
			}

			logger.Info("ws_passenger_message", string(msg), requestID, "")
			conn.WriteMessage(messageType, []byte("Server received: "+string(msg)))
		}
	}
}
