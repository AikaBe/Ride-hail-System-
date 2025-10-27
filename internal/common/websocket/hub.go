package websocket

import (
	"encoding/json"
	"log"
	"ride-hail/internal/common/rmq"
	DriverModel "ride-hail/internal/driver/model"
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
	Clients            map[string]*Client
	Mu                 sync.RWMutex
	Register           chan *Client
	Unregister         chan *Client
	Broadcast          chan []byte
	DriverResponses    chan DriverModel.DriverResponceWS
	PassengerResponses chan rmq.PassiNFO
	UpdateLocation     chan rmq.LocationUpdateMessage
}

func NewHub() *Hub {
	return &Hub{
		Clients:            make(map[string]*Client),
		Register:           make(chan *Client),
		Unregister:         make(chan *Client),
		Broadcast:          make(chan []byte),
		DriverResponses:    make(chan DriverModel.DriverResponceWS, 10),
		PassengerResponses: make(chan rmq.PassiNFO, 10),
		UpdateLocation:     make(chan rmq.LocationUpdateMessage, 10),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mu.Lock()
			h.Clients[client.ID] = client
			h.Mu.Unlock()
		case client := <-h.Unregister:
			h.Mu.Lock()
			delete(h.Clients, client.ID)
			h.Mu.Unlock()
		case message := <-h.Broadcast:
			h.Mu.RLock()
			for _, c := range h.Clients {
				select {
				case c.Send <- message:
				default:
					log.Printf("⚠️ Client %s send buffer full", c)
				}
			}
			h.Mu.RUnlock()
		}
	}
}

func (h *Hub) SendToClient(clientID string, message []byte) {
	h.Mu.RLock()
	client, ok := h.Clients[clientID]
	h.Mu.RUnlock()
	if ok {
		select {
		case client.Send <- message:
			log.Printf("✅ Сообщение отправлено клиенту %s: %s", clientID, string(message))
		default:
			log.Printf("⚠️ Канал переполнен, отключаем клиента %s", clientID)
			go func() {
				h.Unregister <- client
			}()
		}
	} else {
		log.Printf("❌ Клиент %s не найден в Hub", clientID)
	}
}

func (h *Hub) BroadcastRideOffer(msg rmq.RideRequestedMessage) {
	data, _ := json.Marshal(msg)

	h.Mu.RLock()
	defer h.Mu.RUnlock()

	for _, client := range h.Clients {
		if strings.HasPrefix(client.ID, "driver_") {
			select {
			case client.Send <- data:
				log.Printf("📨 Ride offer sent to driver %s for ride %s", client.ID, msg.RideID)
			default:
				log.Printf("⚠️ Channel full, disconnecting driver %s", client.ID)
				go func(c *Client) { h.Unregister <- c }(client)
			}
		}
	}
}

func (h *Hub) ListenDriverMessages(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения от %s: %v", client.ID, err)
			return
		}

		var resp DriverModel.DriverResponceWS
		if err := json.Unmarshal(msg, &resp); err == nil {
			resp.DriverID = client.ID // на всякий случай
			h.DriverResponses <- resp // отправляем в канал ответов
			log.Printf("📩 Ответ от водителя %s: %+v", client.ID, resp)
		} else {
			log.Printf("⚠️ Не удалось распарсить сообщение от %s: %s", client.ID, msg)
		}
	}
}

func (h *Hub) ListenPassengerMessages(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения от %s: %v", client.ID, err)
			return
		}

		var resp rmq.PassiNFO
		if err := json.Unmarshal(msg, &resp); err == nil {
			h.PassengerResponses <- resp // отправляем в канал ответов
			log.Printf("📩 Ответ от водителя %s: %+v", client.ID, resp)
		} else {
			log.Printf("⚠️ Не удалось распарсить сообщение от %s: %s", client.ID, msg)
		}
	}
}

func (h *Hub) UpdateLocationWS(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения от %s: %v", client.ID, err)
			return
		}

		var resp rmq.LocationUpdateMessage
		if err := json.Unmarshal(msg, &resp); err == nil {
			h.UpdateLocation <- resp
			log.Printf("📩 Ответ от водителя %s: %+v", client.ID, resp)
		} else {
			log.Printf("⚠️ Не удалось распарсить сообщение от %s: %s", client.ID, msg)
		}
	}
}
