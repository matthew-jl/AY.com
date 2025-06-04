package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	// For Notification struct
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins for now
}

type Client struct {
	ID     string
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[string]*Client       // Registered clients by client ID
	userClients map[uint]map[string]bool // Map UserID to a set of their client IDs
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte // Not used for direct user messages
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		userClients: make(map[uint]map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		// broadcast:  make(chan []byte), // Not directly used for user-specific
	}
}

func (h *Hub) Run() {
	log.Println("WebSocket Hub running...")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if _, ok := h.userClients[client.UserID]; !ok {
				h.userClients[client.UserID] = make(map[string]bool)
			}
			h.userClients[client.UserID][client.ID] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s (User: %d)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if userConns, userOk := h.userClients[client.UserID]; userOk {
					delete(userConns, client.ID)
					if len(userConns) == 0 {
						delete(h.userClients, client.UserID)
					}
				}
				close(client.Send)
				log.Printf("Client unregistered: %s (User: %d)", client.ID, client.UserID)
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastToUser sends a message to all connected clients of a specific user.
// The message should be a marshaled JSON of the notification.
func (h *Hub) BroadcastToUser(userID uint, notificationData interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

    var jsonData []byte
	var err error
	
	jsonData, err = json.Marshal(notificationData)
    if err != nil {
        log.Printf("BroadcastToUser: Error marshalling notification data to JSON for user %d: %v", userID, err)
		jsonData = []byte(`{"error": "Failed to serialize notification", "message": "You have a new notification."}`)
    }

	if userClientIDs, ok := h.userClients[userID]; ok {
		log.Printf("Broadcasting notification (JSON) to user %d (%d clients): %s", userID, len(userClientIDs), string(jsonData))
		for clientID := range userClientIDs {
			if client, clientOk := h.clients[clientID]; clientOk {
				select {
				case client.Send <- jsonData:
					log.Printf("Sent message to client %s for user %d", client.ID, userID)
				default:
					log.Printf("Client %s send buffer full, closing and unregistering.", client.ID)
					close(client.Send)
				}
			}
		}
	} else {
		log.Printf("No active WebSocket clients for user %d to send notification.", userID)
	}
}


// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID uint) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	clientID := uuid.NewString()
	client := &Client{
		ID:     clientID,
		UserID: userID, // Associate client with authenticated user
		Conn:   conn,
		Send:   make(chan []byte, 256), // Buffered channel
	}
	hub.register <- client

	log.Printf("New WebSocket client connected: %s for UserID: %d", client.ID, client.UserID)

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go client.writePump()
	go client.readPump(hub) // Pass hub to handle unregister on read error
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
		log.Printf("WebSocket client readPump closed: %s", c.ID)
	}()
	// c.Conn.SetReadLimit(maxMessageSize) // Optional: set max message size
	// c.Conn.SetReadDeadline(time.Now().Add(pongWait)) // Optional: for pong handling
	// c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket unexpected close error: %v", err)
			} else {
				log.Printf("WebSocket read error for client %s: %v", c.ID, err)
			}
			break
		}
		// Process incoming message from client (e.g., client sending a ping, or other commands)
		log.Printf("Received message from client %s: %s", c.ID, string(message))
		// For now, we don't expect client messages for notifications, but this is where you'd handle them.
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	// ticker := time.NewTicker(pingPeriod) // Optional: for sending pings
	defer func() {
		// ticker.Stop()
		c.Conn.Close()
		log.Printf("WebSocket client writePump closed: %s", c.ID)
	}()
	for {
		select {
		case message, ok := <-c.Send:
			// c.Conn.SetWriteDeadline(time.Now().Add(writeWait)) // Optional: set write deadline
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Printf("Hub closed send channel for client %s", c.ID)
				return
			}
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("WebSocket write error for client %s: %v", c.ID, err)
				return
			}
		}
	}
}