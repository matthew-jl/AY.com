package http

import (
	"net/http"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	"github.com/gin-gonic/gin"
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