package websocket

import (
	"context"
	"net/http"
	"ride-hail/internal/common/logger"
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
		logger.Error("driver_ws_upgrade", "WebSocket upgrade failed", "", "", err.Error())
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		conn.Close()
	}()

	done := make(chan struct{})

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}

	if err := conn.ReadJSON(&authMsg); err != nil {
		logger.Error("driver_ws_auth", "failed to read auth message", "", "", err.Error())
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "auth failed"))
		return
	}

	claims, err := jwtManager.ValidateToken(authMsg.Token)
	if err != nil {
		logger.Warn("driver_ws_token", "invalid token for driver", "", "", err.Error())
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "invalid token"))
		return
	}

	client := &commonws.Client{
		ID:   "driver_" + claims.UserID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	logger.Info("driver_ws_connect", "driver connected", claims.UserID, "")

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
					logger.Warn("driver_ws_ping", "ping failed", claims.UserID, "", err.Error())
					return
				}
			}
		}
	}()

	go func() {
		for msg := range client.Send {
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Error("driver_ws_send", "failed to send message to driver", claims.UserID, "", err.Error())
				break
			}
		}
	}()

	go hub.ListenDriverMessages(client)
	go svc.SendToMq(context.Background())
	go svc.UpdateLocationWS(context.Background())

	<-done
	hub.Unregister <- client
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	logger.Info("driver_ws_disconnect", "driver disconnected", claims.UserID, "")
}
