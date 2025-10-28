package websocket

import (
	"encoding/json"
	"log"
	"reflect"
	"ride-hail/internal/common/rmq"
	DriverModel "ride-hail/internal/driver/model"
	"strconv"
	"strings"
	"sync"
	"time"

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
		DriverResponses:    make(chan DriverModel.DriverResponceWS, 100),
		PassengerResponses: make(chan rmq.PassiNFO, 100),
		UpdateLocation:     make(chan rmq.LocationUpdateMessage, 100),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mu.Lock()
			h.Clients[client.ID] = client
			h.Mu.Unlock()
			log.Printf("✅ Client registered: %s", client.ID)
		case client := <-h.Unregister:
			h.Mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
			}
			h.Mu.Unlock()
			log.Printf("🚪 Client unregistered: %s", client.ID)
		case message := <-h.Broadcast:
			h.Mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
					// Message sent successfully
				default:
					log.Printf("⚠️ Client %s send buffer full, closing", client.ID)
					h.Unregister <- client
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
			log.Printf("✅ Message sent to client %s", clientID)
		default:
			log.Printf("⚠️ Channel full for client %s, unregistering", clientID)
			h.Unregister <- client
		}
	} else {
		log.Printf("❌ Client %s not found", clientID)
	}
}

func (h *Hub) BroadcastRideOffer(msg rmq.RideRequestedMessage) {
	data, err := json.Marshal(map[string]interface{}{
		"type":                 "ride_offer",
		"ride_id":              msg.RideID,
		"ride_number":          msg.RideNumber,
		"pickup_location":      msg.PickupLocation,
		"destination_location": msg.DestinationLocation,
		"ride_type":            msg.RideType,
		"estimated_fare":       msg.EstimatedFare,
		"timeout_seconds":      msg.TimeoutSeconds,
	})
	if err != nil {
		log.Printf("❌ Failed to marshal ride offer: %v", err)
		return
	}

	h.Mu.RLock()
	defer h.Mu.RUnlock()

	sentCount := 0
	for id, client := range h.Clients {
		if strings.HasPrefix(id, "driver_") {
			select {
			case client.Send <- data:
				sentCount++
			default:
				log.Printf("⚠️ Channel full for driver %s", id)
				go func(c *Client) { h.Unregister <- c }(client)
			}
		}
	}
	log.Printf("📨 Ride offer broadcast to %d drivers for ride %s", sentCount, msg.RideID)
}

func (h *Hub) ListenDriverMessages(client *Client) {
	defer func() {
		h.Unregister <- client
		client.Conn.Close()
	}()

	for {
		var msg map[string]interface{}
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Driver %s disconnected unexpectedly: %v", client.ID, err)
			}
			break
		}

		log.Printf("📨 Raw message from driver %s: %+v", client.ID, msg)

		// Handle location updates
		if msgType, ok := msg["type"].(string); ok && msgType == "location_update" {
			var locUpdate rmq.LocationUpdateMessage
			// Convert the message to LocationUpdateMessage
			if driverID, ok := msg["driver_id"].(string); ok {
				locUpdate.DriverID = strings.TrimPrefix(driverID, "driver_")
			} else {
				locUpdate.DriverID = strings.TrimPrefix(client.ID, "driver_")
			}

			if rideID, ok := msg["ride_id"].(string); ok {
				locUpdate.RideID = rideID
			}

			if location, ok := msg["location"].(map[string]interface{}); ok {
				if lat, ok := location["lat"].(float64); ok {
					locUpdate.Location.Lat = lat
				}
				if lng, ok := location["lng"].(float64); ok {
					locUpdate.Location.Lng = lng
				}
			}

			if speed, ok := msg["speed_kmh"].(float64); ok {
				locUpdate.SpeedKmh = speed
			}

			if heading, ok := msg["heading_degrees"].(float64); ok {
				locUpdate.Heading = heading
			}

			nowMs := time.Now().UnixNano() / int64(time.Millisecond)
			setTimestamp(&locUpdate, nowMs)

			h.UpdateLocation <- locUpdate
			log.Printf("📍 Location update from driver %s", locUpdate.DriverID)
			continue
		}

		// Handle ride responses
		if msgType, ok := msg["type"].(string); ok && msgType == "ride_response" {
			var resp DriverModel.DriverResponceWS
			resp.Type = msgType

			if offerID, ok := msg["offer_id"].(string); ok {
				resp.OfferID = offerID
			}
			if rideID, ok := msg["ride_id"].(string); ok {
				resp.RideID = rideID
			}
			if accepted, ok := msg["accepted"].(bool); ok {
				resp.Accepted = accepted
			}

			resp.DriverID = strings.TrimPrefix(client.ID, "driver_")

			if currentLoc, ok := msg["current_location"].(map[string]interface{}); ok {
				if lat, ok := currentLoc["latitude"].(float64); ok {
					resp.CurrentLocation.Latitude = lat
				}
				if lng, ok := currentLoc["longitude"].(float64); ok {
					resp.CurrentLocation.Longitude = lng
				}
			}

			h.DriverResponses <- resp
			log.Printf("📩 Ride response from driver %s: accepted=%v", resp.DriverID, resp.Accepted)
			continue
		}

		log.Printf("⚠️ Unrecognized message from driver %s: %+v", client.ID, msg)
	}
}

func (h *Hub) ListenPassengerMessages(client *Client) {
	defer func() {
		h.Unregister <- client
		client.Conn.Close()
	}()

	for {
		var msg map[string]interface{}
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Passenger %s disconnected unexpectedly: %v", client.ID, err)
			}
			break
		}

		log.Printf("📨 Message from passenger %s: %+v", client.ID, msg)

		// Handle passenger responses
		if msgType, ok := msg["type"].(string); ok {
			var resp rmq.PassiNFO
			resp.Type = msgType

			if rideID, ok := msg["ride_id"].(string); ok {
				resp.RideID = rideID
			}
			if name, ok := msg["passenger_name"].(string); ok {
				resp.PassengerName = name
			}
			if phone, ok := msg["passenger_phone"].(string); ok {
				resp.PassengerPhone = phone
			}

			if pickup, ok := msg["pickup_location"].(map[string]interface{}); ok {
				if lat, ok := pickup["latitude"].(float64); ok {
					resp.PickupLocation.Latitude = lat
				}
				if lng, ok := pickup["longitude"].(float64); ok {
					resp.PickupLocation.Longitude = lng
				}
				if addr, ok := pickup["address"].(string); ok {
					resp.PickupLocation.Address = addr
				}
				if notes, ok := pickup["notes"].(string); ok {
					resp.PickupLocation.Notes = notes
				}
			}

			h.PassengerResponses <- resp
			log.Printf("📩 Passenger info from %s for ride %s", client.ID, resp.RideID)
		}
	}
}

func setTimestamp(msg interface{}, tsMillis int64) {
	v := reflect.ValueOf(msg)
	// ожидаем указатель на struct
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	f := v.FieldByName("Timestamp")
	if !f.IsValid() || !f.CanSet() {
		return
	}

	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(tsMillis)
	case reflect.String:
		f.SetString(strconv.FormatInt(tsMillis, 10))
	default:
		// попробовать установить json.Number
		jsonNumType := reflect.TypeOf(json.Number(""))
		if f.Type() == jsonNumType {
			f.Set(reflect.ValueOf(json.Number(strconv.FormatInt(tsMillis, 10))))
		}
	}
}
