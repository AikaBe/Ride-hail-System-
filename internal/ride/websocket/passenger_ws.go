package websocket

import (
	"context"
	"net/http"
	"time"

	"ride-hail/internal/common/logger"
	commonws "ride-hail/internal/common/websocket"
	"ride-hail/internal/ride/service"
	"ride-hail/internal/user/jwt"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func PassengerWSHandler(w http.ResponseWriter, r *http.Request, hub *commonws.Hub, jwtManager *jwt.Manager, svc *service.RideService) {
	action := "PassengerWSHandler"
	requestID := ""
	rideID := ""

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(action, "WebSocket upgrade failed", requestID, rideID, err.Error())
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	done := make(chan struct{})

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	logger.Info(action, "Waiting for passenger auth message", requestID, rideID)

	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&authMsg); err != nil {
		logger.Error(action, "Failed to read passenger auth message", requestID, rideID, err.Error())
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "auth failed"))
		return
	}

	claims, err := jwtManager.ValidateToken(authMsg.Token)
	if err != nil {
		logger.Warn(action, "Invalid passenger token", requestID, rideID, err.Error())
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "invalid token"))
		return
	}

	client := &commonws.Client{
		ID:   "passenger_" + claims.UserID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	hub.Register <- client

	logger.Info(action, "Passenger connected: "+claims.UserID, requestID, rideID)

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
					logger.Warn(action, "Ping failed for passenger "+claims.UserID, requestID, rideID, err.Error())
					return
				}
				logger.Debug(action, "Ping sent to passenger "+claims.UserID, requestID, rideID)
			}
		}
	}()

	go func() {
		for msg := range client.Send {
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Error(action, "Failed to send message to passenger "+client.ID, requestID, rideID, err.Error())
				break
			}
			logger.Debug(action, "Message sent to passenger "+client.ID, requestID, rideID)
		}
	}()

	go hub.ListenPassengerMessages(client)
	go svc.SendPassInfo(context.Background())

	<-done
	hub.Unregister <- client
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))

	logger.Info(action, "Passenger connection closed: "+claims.UserID, requestID, rideID)
}
