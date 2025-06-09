package client

import (
	"context"
	"fmt"

	messagepb "github.com/Acad600-TPA/WEB-MJ-242/backend/message-service/genproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type MessageClient struct {
	client messagepb.MessageServiceClient
	conn   *grpc.ClientConn
}

func NewMessageClient(addr string) (*MessageClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return nil, fmt.Errorf("failed to dial message service: %w", err) }
	client := messagepb.NewMessageServiceClient(conn)
	return &MessageClient{client: client, conn: conn}, nil
}

func (c *MessageClient) Close() error {
	if c.conn != nil { return c.conn.Close() }
	return nil
}

func (c *MessageClient) GetOrCreateDirectChat(ctx context.Context, req *messagepb.GetOrCreateDirectChatRequest) (*messagepb.Chat, error) {
	return c.client.GetOrCreateDirectChat(ctx, req)
}
func (c *MessageClient) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.Message, error) {
	return c.client.SendMessage(ctx, req)
}
func (c *MessageClient) GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error) {
	return c.client.GetMessages(ctx, req)
}
func (c *MessageClient) DeleteMessage(ctx context.Context, req *messagepb.DeleteMessageRequest) (*emptypb.Empty, error) {
	return c.client.DeleteMessage(ctx, req)
}
func (c *MessageClient) GetUserChats(ctx context.Context, req *messagepb.GetUserChatsRequest) (*messagepb.GetUserChatsResponse, error) {
	return c.client.GetUserChats(ctx, req)
}
func (c *MessageClient) HealthCheck(ctx context.Context) (*messagepb.HealthResponse, error) {
    return c.client.HealthCheck(ctx, &emptypb.Empty{})
}
func (c *MessageClient) DeleteChat(ctx context.Context, req *messagepb.DeleteChatRequest) (*emptypb.Empty, error) {
	return c.client.DeleteChat(ctx, req)
}
func (c *MessageClient) CreateGroupChat(ctx context.Context, req *messagepb.CreateGroupChatRequest) (*messagepb.Chat, error) {
	return c.client.CreateGroupChat(ctx, req)
}
func (c *MessageClient) AddParticipantToGroup(ctx context.Context, req *messagepb.UpdateGroupParticipantsRequest) (*emptypb.Empty, error) {
	return c.client.AddParticipantToGroup(ctx, req)
}
func (c *MessageClient) RemoveParticipantFromGroup(ctx context.Context, req *messagepb.UpdateGroupParticipantsRequest) (*emptypb.Empty, error) {
	return c.client.RemoveParticipantFromGroup(ctx, req)
}