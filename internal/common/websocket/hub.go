package websocket

import (
	"encoding/json"
	"log"
	"ride-hail/internal/common/rmq"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}
type Hub struct {
	Clients         map[string]*Client
	mu              sync.RWMutex
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
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			delete(h.Clients, client.ID)
			close(client.Send)
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, c := range h.Clients {
				select {
				case c.Send <- message:
				default:
					close(c.Send)
					delete(h.Clients, c.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}
func (h *Hub) SendToClient(clientID string, message []byte) {
	h.mu.RLock()
	client, ok := h.Clients[clientID]
	h.mu.RUnlock()
	if ok {
		select {
		case client.Send <- message:
			log.Printf("✅ Сообщение отправлено клиенту %s: %s", clientID, string(message))
		default:
			log.Printf("⚠️ Канал переполнен, отключаем клиента %s", clientID)
			h.mu.Lock()
			close(client.Send)
			delete(h.Clients, clientID)
			h.mu.Unlock()
		}
	} else {
		log.Printf("❌ Клиент %s не найден в Hub", clientID)
	}
}

func (h *Hub) BroadcastRideOffer(msg rmq.RideRequestedMessage) {
	data, _ := json.Marshal(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.Clients {
		if strings.HasPrefix(client.ID, "driver_") {
			select {
			case client.Send <- data:
				log.Printf("📨 Ride offer sent to driver %s for ride %s", client.ID, msg.RideID)
			default:
				log.Printf("⚠️ Channel full, disconnecting driver %s", client.ID)
				close(client.Send)
				h.mu.RUnlock()
				h.mu.Lock()
				delete(h.Clients, client.ID)
				h.mu.Unlock()
				h.mu.RLock()
			}
		}
	}
}

func (h *Hub) ListenClientMessages(client *Client) {
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
