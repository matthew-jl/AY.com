package http

import (
	"net/http"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	userClient *client.UserClient
}

func NewAuthHandler(userClient *client.UserClient) *AuthHandler {
	return &AuthHandler{userClient: userClient}
}

func (h *AuthHandler) HealthCheck(c *gin.Context) {
	resp, err := h.userClient.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": resp.Status})
}

// Register forwards the registration request to the user service
func (h *AuthHandler) Register(c *gin.Context) {
	var req userpb.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Forward request to User Service via gRPC client
	resp, err := h.userClient.Register(c.Request.Context(), &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			c.JSON(httpCode, gin.H{"error": st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User service registration failed: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req userpb.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Check if email or code is missing after binding
	if req.Email == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and code are required"})
		return
	}


	_, err := h.userClient.VerifyEmail(c.Request.Context(), &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			c.JSON(httpCode, gin.H{"error": st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User service verification failed: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account verified successfully"})
}

// Login forwards the login request to the user service
func (h *AuthHandler) Login(c *gin.Context) {
	var req userpb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	resp, err := h.userClient.Login(c.Request.Context(), &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			c.JSON(httpCode, gin.H{"error": st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User service login failed: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}


// Helper function to map gRPC status codes to HTTP status codes
func grpcStatusCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}