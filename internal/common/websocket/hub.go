package websocket

import (
	"encoding/json"
	"strings"
	"sync"

	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/rmq"
	DriverModel "ride-hail/internal/driver/model"

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
	logger.SetServiceName("websocket-hub")

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
			logger.Info("client_register", "Client connected", "", client.ID)

		case client := <-h.Unregister:
			h.Mu.Lock()
			delete(h.Clients, client.ID)
			h.Mu.Unlock()
			logger.Info("client_unregister", "Client disconnected", "", client.ID)

		case message := <-h.Broadcast:
			h.Mu.RLock()
			for _, c := range h.Clients {
				select {
				case c.Send <- message:
				default:
					logger.Warn("broadcast", "Client send buffer full", "", c.ID, "send channel full")
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
			logger.Info("send_to_client", "Message sent to client", "", clientID)
		default:
			logger.Warn("send_to_client", "Client channel full, unregistering", "", clientID, "send channel full")
			go func() {
				h.Unregister <- client
			}()
		}
	} else {
		logger.Warn("send_to_client", "Client not found in Hub", "", clientID, "not found")
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
				logger.Info("broadcast_ride_offer", "Ride offer sent to driver", msg.RideID, client.ID)
			default:
				logger.Warn("broadcast_ride_offer", "Driver channel full, disconnecting", msg.RideID, client.ID, "channel full")
				go func(c *Client) { h.Unregister <- c }(client)
			}
		}
	}
}

func (h *Hub) ListenDriverMessages(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			logger.Error("driver_ws_read", "Error reading from driver", "", client.ID, err.Error())
			return
		}

		var loc rmq.LocationUpdateMessage
		if err := json.Unmarshal(msg, &loc); err == nil && loc.DriverID != "" && loc.RideID != "" {
			if strings.HasPrefix(loc.DriverID, "driver_") {
				loc.DriverID = strings.TrimPrefix(loc.DriverID, "driver_")
			}
			h.UpdateLocation <- loc
			logger.Info("driver_location", "Received location update", loc.RideID, loc.DriverID)
			continue
		}

		var resp DriverModel.DriverResponceWS
		if err := json.Unmarshal(msg, &resp); err == nil && resp.Type == "ride_response" {
			resp.DriverID = client.ID
			if strings.HasPrefix(resp.DriverID, "driver_") {
				resp.DriverID = strings.TrimPrefix(resp.DriverID, "driver_")
			}
			h.DriverResponses <- resp
			logger.Info("driver_response", "Received driver response", "", resp.DriverID)
			continue
		}

		logger.Warn("driver_ws_message", "Unrecognized message from driver", "", client.ID, string(msg))
	}
}

func (h *Hub) ListenPassengerMessages(client *Client) {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			logger.Error("passenger_ws_read", "Error reading from passenger", "", client.ID, err.Error())
			return
		}

		var resp rmq.PassiNFO
		if err := json.Unmarshal(msg, &resp); err == nil {
			h.PassengerResponses <- resp
			logger.Info("passenger_response", "Received passenger response", "", client.ID)
		} else {
			logger.Warn("passenger_ws_message", "Failed to parse passenger message", "", client.ID, string(msg))
		}
	}
}
