// backend/message-service/websocket/hub.go
package websocket

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time" // For ping/pong and potential message timestamps

	messagepb "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/genproto/proto" // For message struct
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait       = 60 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
	maxMessageSize = 1024 * 4            // Maximum message size allowed from peer (e.g., 4KB for control messages)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins for dev
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID     string          // Unique ID for this client connection
	UserID uint            // Authenticated user ID
	Hub    *Hub            // Reference to the hub
	Conn   *websocket.Conn // The websocket connection.
	Send   chan []byte     // Buffered channel of outbound messages.
	// Subscriptions map[uint]bool // Set of ChatIDs this client is subscribed to
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients    map[string]*Client         // Registered clients by their unique connection ID.
	chatRooms  map[uint]map[string]*Client // Map ChatID to a set of clients in that room (using client.ID as key for set).
	register   chan *Client
	unregister chan *Client
	// incoming   chan InboundMessage // Channel for messages from clients (if clients send more than pings)
	// outgoingToChat chan OutboundChatMessage // Channel for messages from gRPC handler to be sent to a chat
	mu sync.RWMutex // Protects clients and chatRooms maps
}

// InboundMessage might be used if clients send structured messages to the server via WS
// type InboundMessage struct {
//  Client *Client
//  Raw    []byte
//  Type   string // e.g., "subscribe_chat", "unsubscribe_chat", "typing_indicator"
//  Data   json.RawMessage
// }

type OutboundChatMessage struct {
    ChatID  uint
    Message *messagepb.Message // The gRPC message to send
}

type MessageUpdateEvent struct {
	Type      string      `json:"type"`       // e.g., "message_deleted", "message_edited", "read_receipt"
	ChatID    uint32      `json:"chat_id"`
	MessageID uint32      `json:"message_id"`
	ActorID   uint32      `json:"actor_id,omitempty"`
}


func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		chatRooms:  make(map[uint]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		// incoming:   make(chan InboundMessage),
		// outgoingToChat: make(chan OutboundChatMessage, 256), // Buffered
	}
}

func (h *Hub) Run() {
	log.Println("Message WebSocket Hub running...")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			log.Printf("Client registered to Hub: %s (User: %d)", client.ID, client.UserID)
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				// Remove from all chat rooms this client was part of
				for chatID, roomClients := range h.chatRooms {
					if _, clientInRoom := roomClients[client.ID]; clientInRoom {
						delete(roomClients, client.ID)
						if len(roomClients) == 0 { // If room is empty, delete it
							delete(h.chatRooms, chatID)
							log.Printf("Chat room %d is now empty and removed.", chatID)
						}
					}
				}
				delete(h.clients, client.ID)
				close(client.Send)
				log.Printf("Client unregistered from Hub: %s (User: %d)", client.ID, client.UserID)
			}
			h.mu.Unlock()

		// Example: Processing messages sent from gRPC handler to be broadcasted
		// case chatMessage := <-h.outgoingToChat:
		//  h.broadcastMessageToChatInternal(chatMessage.ChatID, chatMessage.Message)
		}
	}
}

// SubscribeClientToChat adds a client to a specific chat room.
// This would be called when a client indicates they are viewing/active in a chat.
func (h *Hub) SubscribeClientToChat(client *Client, chatID uint) {
	if client == nil { return }
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.chatRooms[chatID]; !ok {
		h.chatRooms[chatID] = make(map[string]*Client)
	}
	h.chatRooms[chatID][client.ID] = client
	log.Printf("Client %s (User %d) subscribed to ChatID %d", client.ID, client.UserID, chatID)
}

// UnsubscribeClientFromChat removes a client from a chat room.
func (h *Hub) UnsubscribeClientFromChat(client *Client, chatID uint) {
	if client == nil { return }
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomClients, ok := h.chatRooms[chatID]; ok {
		delete(roomClients, client.ID)
		if len(roomClients) == 0 {
			delete(h.chatRooms, chatID)
			log.Printf("Chat room %d is now empty and removed after client %s unsubscribed.", chatID, client.ID)
		}
	}
	log.Printf("Client %s (User %d) unsubscribed from ChatID %d", client.ID, client.UserID, chatID)
}

// BroadcastMessageToChat sends a marshaled message to all clients subscribed to a specific chat.
// This method is called by the gRPC SendMessage handler.
func (h *Hub) BroadcastMessageToChat(chatID uint, message *messagepb.Message) {
	jsonData, err := json.Marshal(message) // Marshal the proto message to JSON
	if err != nil {
		log.Printf("Hub: Error marshalling message for ChatID %d: %v", chatID, err)
		return
	}

	h.mu.RLock() // Use RLock for reading chatRooms and clients
	defer h.mu.RUnlock()

	if roomClients, ok := h.chatRooms[chatID]; ok {
		log.Printf("Hub: Broadcasting message (ID: %d) to %d clients in ChatID %d", message.Id, len(roomClients), chatID)
		for clientID := range roomClients {
			if client, clientOk := h.clients[clientID]; clientOk {
				select {
				case client.Send <- jsonData:
					log.Printf("Hub: Sent message to client %s (User %d) in ChatID %d", client.ID, client.UserID, chatID)
				default:
					log.Printf("Hub: Client %s send buffer full for ChatID %d. Consider closing.", client.ID, chatID)
					// Don't close/unregister from here directly to avoid deadlock on h.mu
					// The client's writePump will handle closure if writes fail.
				}
			}
		}
	} else {
		log.Printf("Hub: No clients currently subscribed to ChatID %d for message ID %d.", chatID, message.Id)
	}
}

func (h *Hub) BroadcastMessageUpdateToChat(chatID uint, updateEvent MessageUpdateEvent) {
	jsonData, err := json.Marshal(updateEvent)
	if err != nil {
		log.Printf("Hub: Error marshalling message update for ChatID %d: %v", chatID, err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if roomClients, ok := h.chatRooms[chatID]; ok {
		log.Printf("Hub: Broadcasting message UPDATE (Type: %s, MsgID: %d) to %d clients in ChatID %d",
			updateEvent.Type, updateEvent.MessageID, len(roomClients), chatID)
		for clientID := range roomClients {
			if client, clientOk := h.clients[clientID]; clientOk {
				select {
				case client.Send <- jsonData:
				default:
					log.Printf("Hub: Client %s send buffer full for message UPDATE in ChatID %d. Consider closing.", client.ID, chatID)
				}
			}
		}
	} else {
		log.Printf("Hub: No clients subscribed to ChatID %d for message update (MsgID: %d).", chatID, updateEvent.MessageID)
	}
}


// ServeWs handles websocket requests from the peer.
// userID is the authenticated user ID from the HTTP request.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID uint) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error for User %d: %v", userID, err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	clientID := uuid.NewString()
	client := &Client{
		ID:     clientID,
		UserID: userID,
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256), // Buffered channel
	}
	client.Hub.register <- client

	log.Printf("New WebSocket client connected: %s for UserID: %d", client.ID, client.UserID)

	go client.writePump()
	go client.readPump() // Pass hub to handle unregister on read error
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
		log.Printf("WebSocket client readPump closed: %s (User: %d)", c.ID, c.UserID)
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait)) // Initial read deadline
	c.Conn.SetPongHandler(func(string) error {
		// log.Printf("Pong received from client %s", c.ID)
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("WebSocket unexpected close error for client %s: %v", c.ID, err)
			} else if err != io.EOF { // Don't log EOF as a verbose error
				log.Printf("WebSocket read error for client %s: %v", c.ID, err)
			}
			break
		}
		// Process incoming message from client
		// Example: Client sends a JSON message to subscribe/unsubscribe from a chat
		// { "type": "subscribe_chat", "chat_id": 123 }
		// { "type": "unsubscribe_chat", "chat_id": 123 }
		// { "type": "typing_indicator", "chat_id": 123, "is_typing": true }
		log.Printf("Received raw message from client %s (User %d): %s", c.ID, c.UserID, string(message))

        var clientMsg struct {
            Type   string          `json:"type"`
            ChatID uint            `json:"chat_id"`
            // Add other fields as needed e.g. for typing indicators
        }
        if err := json.Unmarshal(message, &clientMsg); err == nil {
            switch clientMsg.Type {
            case "subscribe_chat":
                c.Hub.SubscribeClientToChat(c, clientMsg.ChatID)
            case "unsubscribe_chat":
                c.Hub.UnsubscribeClientFromChat(c, clientMsg.ChatID)
            // Handle other message types like "typing_indicator"
            default:
                log.Printf("Client %s sent unknown message type: %s", c.ID, clientMsg.Type)
            }
        } else {
            log.Printf("Client %s sent unparseable message: %s, Error: %v", c.ID, string(message), err)
        }

		// If not processing structured messages, this can be simpler:
		// log.Printf("Received message from client %s (User %d): %s - Ignoring.", c.ID, c.UserID, string(message))
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close() // Ensure connection is closed if writePump exits
		log.Printf("WebSocket client writePump closed: %s (User: %d)", c.ID, c.UserID)
		// Unregister might have already happened in readPump, or hub might do it if send fails
		// c.Hub.unregister <- c // Be careful with unregistering from multiple places
	}()
	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel c.Send.
				log.Printf("Hub closed send channel for client %s, sending close message.", c.ID)
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for client %s: %v", c.ID, err)
				return
			}
			_, err = w.Write(message)
			if err != nil {
				log.Printf("Error writing message to client %s: %v", c.ID, err)
				// No return here, let the loop continue, readPump will handle broken pipe
			}

			// Add queued chat messages to the current websocket message.
			// n := len(c.Send)
			// for i := 0; i < n; i++ {
			// 	w.Write([]byte{'\n'}) // If sending multiple JSON objects, newline delimit them
			// 	w.Write(<-c.Send)
			// }

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer for client %s: %v", c.ID, err)
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping to client %s: %v", c.ID, err)
				return // Assume connection is broken
			}
			// log.Printf("Ping sent to client %s", c.ID)
		}
	}
}