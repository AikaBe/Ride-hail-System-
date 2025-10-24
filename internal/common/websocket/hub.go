package websocket

import (
	"encoding/json"
	"log"
	"ride-hail/internal/common/rmq"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}

type Hub struct {
	Clients         map[string]*Client
	Register        chan *Client
	Unregister      chan *Client
	Broadcast       chan []byte
	DriverResponses chan rmq.DriverResponseMessage
}

func NewHub() *Hub {
	return &Hub{
		Clients:         make(map[string]*Client),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		Broadcast:       make(chan []byte),
		DriverResponses: make(chan rmq.DriverResponseMessage, 10),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.ID] = client
		case client := <-h.Unregister:
			delete(h.Clients, client.ID)
			close(client.Send)
		case message := <-h.Broadcast:
			for _, c := range h.Clients {
				c.Send <- message
			}
		}
	}
}

func (h *Hub) SendToClient(clientID string, message []byte) {
	client, ok := h.Clients[clientID]
	if ok {
		select {
		case client.Send <- message:
			// сообщение успешно отправлено
		default:
			// если канал переполнен, закрываем соединение
			close(client.Send)
			delete(h.Clients, clientID)
		}
	}
}

func (h *Hub) listenClientMessages(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения от %s: %v", client.ID, err)
			h.Unregister <- client
			return
		}

		var resp rmq.DriverResponseMessage
		if err := json.Unmarshal(msg, &resp); err == nil {
			resp.DriverID = client.ID // на всякий случай
			h.DriverResponses <- resp // отправляем в канал ответов
			log.Printf("📩 Ответ от водителя %s: %+v", client.ID, resp)
		} else {
			log.Printf("⚠️ Не удалось распарсить сообщение от %s: %s", client.ID, msg)
		}
	}
}
