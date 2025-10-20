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

func DriverWSHandler(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws_upgrade_failed", "Failed to upgrade connection", requestID, "", err.Error(), "")
		http.Error(w, "failed to upgrade", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	logger.Info("ws_driver_connected", "Driver connected", requestID, "")

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
		logger.Warn("ws_invalid_token", "Driver token invalid", requestID, "", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid_token"}`))
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(`{"status":"authenticated"}`))
	logger.Info("ws_driver_authenticated", "Driver successfully authenticated", requestID, "")

	conn.SetPongHandler(func(appData string) error {
		logger.Debug("ws_pong_received", "Pong received from driver", requestID, "")
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("ws_ping_failed", "Ping to driver failed", requestID, "", err.Error(), "")
				return
			}

		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Error("ws_read_failed", "Failed to read driver message", requestID, "", err.Error(), "")
				return
			}

			logger.Info("ws_driver_message", string(msg), requestID, "")
			conn.WriteMessage(messageType, []byte("Server received: "+string(msg)))
		}
	}
}
