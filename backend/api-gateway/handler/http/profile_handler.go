// backend/api-gateway/handler/http/profile_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	userClient *client.UserClient
}

func NewProfileHandler(userClient *client.UserClient) *ProfileHandler {
	return &ProfileHandler{userClient: userClient}
}

func (h *ProfileHandler) GetUserProfileByUsername(c *gin.Context) {
	usernameToView := c.Param("username")
	if usernameToView == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username parameter is required"})
		return
	}

	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToView})
	if err != nil {
		handleGRPCError(c, "get target user by username", err)
		return
	}
	if targetUserPb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found"})
		return
	}


	requesterUserID, _ := getUserIDFromContext(c)

	profileResp, err := h.userClient.GetUserProfile(c.Request.Context(), &userpb.GetUserProfileRequest{
		UserIdToView:    targetUserPb.GetId(),
		RequesterUserId: &requesterUserID,
	})
	if err != nil {
		handleGRPCError(c, "get user profile details", err)
		return
	}

	c.JSON(http.StatusOK, profileResp)
}


func (h *ProfileHandler) FollowUser(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	usernameToFollow := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToFollow})
	if err != nil { handleGRPCError(c, "find user to follow", err); return }

	_, err = h.userClient.FollowUser(c.Request.Context(), &userpb.FollowRequest{
		FollowerId: requesterUserID,
		FollowedId: targetUserPb.GetId(),
	})
	if err != nil { handleGRPCError(c, "follow user", err); return }

	c.JSON(http.StatusOK, gin.H{"message": "Successfully followed user"})
}

func (h *ProfileHandler) UnfollowUser(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	usernameToUnfollow := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToUnfollow})
	if err != nil { handleGRPCError(c, "find user to unfollow", err); return }

	_, err = h.userClient.UnfollowUser(c.Request.Context(), &userpb.FollowRequest{
		FollowerId: requesterUserID,
		FollowedId: targetUserPb.GetId(),
	})
	if err != nil { handleGRPCError(c, "unfollow user", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Successfully unfollowed user"})
}

func (h *ProfileHandler) BlockUser(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	usernameToBlock := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToBlock})
	if err != nil { handleGRPCError(c, "find user to block", err); return }

	_, err = h.userClient.BlockUser(c.Request.Context(), &userpb.BlockRequest{
		BlockerId: requesterUserID,
		BlockedId: targetUserPb.GetId(),
	})
	if err != nil { handleGRPCError(c, "block user", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Successfully blocked user"})
}

func (h *ProfileHandler) UnblockUser(c *gin.Context) {
	requesterUserID, ok := getUserIDFromContext(c)
	if !ok { return }

	usernameToUnblock := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: usernameToUnblock})
	if err != nil { handleGRPCError(c, "find user to unblock", err); return }

	_, err = h.userClient.UnblockUser(c.Request.Context(), &userpb.BlockRequest{
		BlockerId: requesterUserID,
		BlockedId: targetUserPb.GetId(),
	})
	if err != nil { handleGRPCError(c, "unblock user", err); return }
	c.JSON(http.StatusOK, gin.H{"message": "Successfully unblocked user"})
}


func (h *ProfileHandler) GetFollowers(c *gin.Context) {
	username := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: username})
	if err != nil { handleGRPCError(c, "find user for followers list", err); return }

	requesterUserID, _ := getUserIDFromContext(c)

	page, limit := parsePagination(c)
	resp, err := h.userClient.GetFollowers(c.Request.Context(), &userpb.GetSocialListRequest{
		UserId:         targetUserPb.GetId(),
		RequesterUserId: &requesterUserID,
		Page:           page,
		Limit:          limit,
	})
	if err != nil { handleGRPCError(c, "get followers", err); return }
	c.JSON(http.StatusOK, resp)
}

func (h *ProfileHandler) GetFollowing(c *gin.Context) {
	username := c.Param("username")
	targetUserPb, err := h.userClient.GetUserByUsername(c.Request.Context(), &userpb.GetUserByUsernameRequest{Username: username})
	if err != nil { handleGRPCError(c, "find user for following list", err); return }

	requesterUserID, _ := getUserIDFromContext(c)

	page, limit := parsePagination(c)
	resp, err := h.userClient.GetFollowing(c.Request.Context(), &userpb.GetSocialListRequest{
		UserId:         targetUserPb.GetId(),
		RequesterUserId: &requesterUserID,
		Page:           page,
		Limit:          limit,
	})
	if err != nil { handleGRPCError(c, "get following", err); return }
	c.JSON(http.StatusOK, resp)
}

// TODO: move to utils
func parsePagination(c *gin.Context) (page, limit int32) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	p, err := strconv.Atoi(pageStr)
	if err != nil || p < 1 { p = 1 }
	l, err := strconv.Atoi(limitStr)
	if err != nil || l < 1 || l > 50 { l = 20 }
	return int32(p), int32(l)
}

