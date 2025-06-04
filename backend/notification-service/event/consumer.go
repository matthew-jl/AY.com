package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/repository/postgres"
	notifUtils "github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/utils"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/notification-service/websocket"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Define expected event structures from RabbitMQ
type UserRegisteredEvent struct {
	UserID   uint   `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type ThreadLikedEvent struct {
	ThreadID  uint   `json:"thread_id"`
	ThreadAuthorID uint `json:"thread_author_id"` // ID of the user whose thread was liked
	LikedByUserID   uint   `json:"liked_by_user_id"`   // ID of the user who liked the thread
    LikedByUsername string `json:"liked_by_username"`
}

type NewFollowerEvent struct {
    FollowedUserID uint   `json:"followed_user_id"` // User who gained a follower
    FollowerUserID uint   `json:"follower_user_id"` // User who started following
    FollowerUsername string `json:"follower_username"`
}

type MentionEvent struct {
    ThreadID uint   `json:"thread_id"`
    MentionedUserID uint `json:"mentioned_user_id"`
    MentioningUserID uint `json:"mentioning_user_id"`
    MentioningUsername string `json:"mentioning_username"`
    ThreadContentSnippet string `json:"thread_content_snippet"`
}


// Define queue and exchange names
const (
	UserEventsExchange  = "user_events"
	UserRegisteredQueue = "user_registered_notif_queue"
	UserRegisteredRoutingKey = "user.registered"

    ThreadEventsExchange = "thread_events"
    ThreadLikedQueue = "thread_liked_notif_queue"
    ThreadLikedRoutingKey = "thread.liked"
    MentionQueue = "mention_notif_queue"
    MentionRoutingKey = "thread.mentioned"

    SocialEventsExchange = "social_events"
    NewFollowerQueue = "new_follower_notif_queue"
    NewFollowerRoutingKey = "social.new_follower"
)


type Consumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	repo         *postgres.NotificationRepository
	userClient   userpb.UserServiceClient
	webSocketHub *websocket.Hub
}

func NewConsumer(repo *postgres.NotificationRepository, uc userpb.UserServiceClient, wsHub *websocket.Hub) (*Consumer, error) {
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		log.Fatalln("RABBITMQ_URL not set")
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil { return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err) }

	ch, err := conn.Channel()
	if err != nil { conn.Close(); return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err) }

	// Declare exchanges (idempotent)
	err = ch.ExchangeDeclare(UserEventsExchange, "topic", true, false, false, false, nil)
	if err != nil { /* handle error */ }
    err = ch.ExchangeDeclare(ThreadEventsExchange, "topic", true, false, false, false, nil)
	if err != nil { /* handle error */ }
    err = ch.ExchangeDeclare(SocialEventsExchange, "topic", true, false, false, false, nil)
	if err != nil { /* handle error */ }


	// Declare queues and bind them
	declareAndBind(ch, UserRegisteredQueue, UserEventsExchange, UserRegisteredRoutingKey)
    declareAndBind(ch, ThreadLikedQueue, ThreadEventsExchange, ThreadLikedRoutingKey)
    declareAndBind(ch, NewFollowerQueue, SocialEventsExchange, NewFollowerRoutingKey)
    declareAndBind(ch, MentionQueue, ThreadEventsExchange, MentionRoutingKey)


	return &Consumer{conn: conn, channel: ch, repo: repo, userClient: uc, webSocketHub: wsHub}, nil
}

func declareAndBind(ch *amqp.Channel, queueName, exchangeName, routingKey string) {
    _, err := ch.QueueDeclare(queueName, true, false, false, false, nil) // Durable queue
    if err != nil { log.Fatalf("Failed to declare queue %s: %v", queueName, err) }
    err = ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
    if err != nil { log.Fatalf("Failed to bind queue %s to exchange %s with key %s: %v", queueName, exchangeName, routingKey, err) }
    log.Printf("Queue %s declared and bound to %s with key %s", queueName, exchangeName, routingKey)
}


func (c *Consumer) StartConsuming() {
	log.Println("Notification Consumer starting...")
	// Consume from different queues
	go c.consume(UserRegisteredQueue, c.handleUserRegistered)
    go c.consume(ThreadLikedQueue, c.handleThreadLiked)
    go c.consume(NewFollowerQueue, c.handleNewFollower)
    go c.consume(MentionQueue, c.handleMention)
}

func (c *Consumer) consume(queueName string, handlerFunc func(d amqp.Delivery)) {
    msgs, err := c.channel.Consume(
        queueName, // queue
        "",        // consumer
        false,     // auto-ack (set to false for manual ack)
        false,     // exclusive
        false,     // no-local
        false,     // no-wait
        nil,       // args
    )
    if err != nil {
        log.Fatalf("Failed to register a consumer for queue %s: %v", queueName, err)
    }

    forever := make(chan bool)
    go func() {
        for d := range msgs {
            log.Printf("Received a message from %s: %s", queueName, truncate(string(d.Body), 100))
            handlerFunc(d) // Process the message
            // Acknowledge message after processing
            if err := d.Ack(false); err != nil {
                log.Printf("Error acknowledging message from %s: %v", queueName, err)
                // Handle ack error (e.g., requeue or move to dead-letter)
            }
        }
    }()
    log.Printf(" [*] Waiting for messages on %s. To exit press CTRL+C", queueName)
    <-forever // Keep the goroutine alive
}

func (c *Consumer) handleUserRegistered(d amqp.Delivery) {
	var event UserRegisteredEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Error unmarshalling UserRegisteredEvent: %v. Body: %s", err, string(d.Body))
		return
	}
	log.Printf("Handling UserRegisteredEvent for UserID: %d, Email: %s", event.UserID, event.Email)
}

func (c *Consumer) handleThreadLiked(d amqp.Delivery) {
	var event ThreadLikedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Error unmarshalling ThreadLikedEvent: %v", err)
		return
	}
	log.Printf("Handling ThreadLikedEvent: ThreadID %d, LikedBy %d (%s), Author %d",
		event.ThreadID, event.LikedByUserID, event.LikedByUsername, event.ThreadAuthorID)

    if event.ThreadAuthorID == event.LikedByUserID { return } // Don't notify for own like

	notificationMsg := fmt.Sprintf("@%s liked your thread.", event.LikedByUsername)
	notif := &postgres.Notification{
		UserID:   event.ThreadAuthorID, // Notify the author of the thread
		Type:     "thread_like",
		Message:  notificationMsg,
		EntityID: fmt.Sprintf("%d", event.ThreadID), // Store thread ID
		ActorID:  &event.LikedByUserID,
	}
	if err := c.repo.CreateNotification(context.Background(), notif); err != nil {
		log.Printf("Failed to save 'thread_like' notification: %v", err)
		return
	}
    log.Printf("Saved 'thread_like' notification for user %d", notif.UserID)

    // Push real-time notification
    c.webSocketHub.BroadcastToUser(notif.UserID, notif) // Assumes Hub has this method

	// Queue/Send email notification
	go c.sendEmailForNotification(notif.UserID, "Someone liked your thread!", notificationMsg)
}

func (c *Consumer) handleNewFollower(d amqp.Delivery) {
    var event NewFollowerEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Error unmarshalling NewFollowerEvent: %v. Body: %s", err, string(d.Body))
		return
	}
    log.Printf("Handling NewFollowerEvent: Followed %d, Follower %d (%s)", event.FollowedUserID, event.FollowerUserID, event.FollowerUsername)

    notificationMsg := fmt.Sprintf("@%s started following you.", event.FollowerUsername)
    notif := &postgres.Notification{
        UserID: event.FollowedUserID, Type: "new_follower", Message: notificationMsg,
        EntityID: fmt.Sprintf("%d", event.FollowerUserID), ActorID: &event.FollowerUserID,
    }
    if err := c.repo.CreateNotification(context.Background(), notif); err != nil {
		log.Printf("Failed to save 'new_follower' notification: %v", err)
		return 
	}
    log.Printf("Saved 'new_follower' notification for user %d", notif.UserID)
    c.webSocketHub.BroadcastToUser(notif.UserID, notif)
    go c.sendEmailForNotification(notif.UserID, "You have a new follower!", notificationMsg)
}

func (c *Consumer) handleMention(d amqp.Delivery) {
    var event MentionEvent
    if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Error unmarshalling MentionEvent: %v. Body: %s", err, string(d.Body))
		return 
	}
    log.Printf("Handling MentionEvent: Thread %d, Mentioned %d, Mentioner %d (%s)",
        event.ThreadID, event.MentionedUserID, event.MentioningUserID, event.MentioningUsername)

    if event.MentionedUserID == event.MentioningUserID { return } // No self-mention notification

    notificationMsg := fmt.Sprintf("@%s mentioned you in a thread: \"%s\"", event.MentioningUsername, truncate(event.ThreadContentSnippet, 50))
    notif := &postgres.Notification{
        UserID: event.MentionedUserID, Type: "mention", Message: notificationMsg,
        EntityID: fmt.Sprintf("%d", event.ThreadID), ActorID: &event.MentioningUserID,
    }
    if err := c.repo.CreateNotification(context.Background(), notif); err != nil {
		log.Printf("Failed to save 'mention' notification: %v", err)
		return 
	}
    log.Printf("Saved 'mention' notification for user %d", notif.UserID)
    c.webSocketHub.BroadcastToUser(notif.UserID, notif)
    go c.sendEmailForNotification(notif.UserID, "You were mentioned in a thread!", notificationMsg)
}


func (c *Consumer) sendEmailForNotification(userID uint, subject, body string) {
    if c.userClient == nil { log.Println("Cannot send email: userClient not configured in consumer"); return }

    // TODO: Check user's notification preferences before sending email
    // For now, assume they want emails for everything.

    userProfileResp, err := c.userClient.GetUserProfile(context.Background(), &userpb.GetUserProfileRequest{UserIdToView: uint32(userID)})
    if err != nil || userProfileResp == nil || userProfileResp.User == nil {
        log.Printf("Failed to get user %d email for notification: %v", userID, err)
        return
    }
    if userProfileResp.User.Email == "" {
        log.Printf("User %d has no email address for notification.", userID)
        return
    }

    err = notifUtils.SendNotificationEmail(userProfileResp.User.Email, subject, body)
    if err != nil {
        log.Printf("Error sending notification email to %s for user %d: %v", userProfileResp.User.Email, userID, err)
    }
}


func (c *Consumer) Close() {
	if c.channel != nil { c.channel.Close() }
	if c.conn != nil { c.conn.Close() }
	log.Println("Notification Consumer stopped.")
}

// Helper for logging
func truncate(s string, maxLen int) string {
    if len(s) <= maxLen { return s }
    return s[:maxLen] + "..."
}