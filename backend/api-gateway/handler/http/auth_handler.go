package http

import (
	"log"
	"net/http"
	"time"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	"github.com/golang-jwt/jwt/v5"

	gwUtils "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/utils"
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

type RegisterPayload struct {
	Name             string `json:"name" binding:"required"`
	Username         string `json:"username" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required"`
	Gender           string `json:"gender"`
	DateOfBirth      string `json:"date_of_birth" binding:"required"`
	SecurityQuestion string `json:"security_question" binding:"required"`
	SecurityAnswer   string `json:"security_answer" binding:"required"`
	RecaptchaToken   string `json:"recaptchaToken" binding:"required"`
	SubscribedToNewsletter bool   `json:"subscribed_to_newsletter"`
	ProfilePictureURL      *string `json:"profile_picture_url,omitempty"`
	BannerURL              *string `json:"banner_url,omitempty"`
}

type LoginPayload struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	RecaptchaToken string `json:"recaptchaToken" binding:"required"`
}

type VerifyEmailPayload struct {
    Email string `json:"email" binding:"required,email"`
    Code  string `json:"code" binding:"required"`
}

type GetSecurityQuestionPayload struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordPayload struct {
	Email           string `json:"email" binding:"required,email"`
	SecurityAnswer  string `json:"security_answer" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type ResendVerificationPayload struct {
    Email string `json:"email" binding:"required,email"`
}

type UpdateProfilePayload struct {
	Name                   *string `json:"name,omitempty"`
	Bio                    *string `json:"bio,omitempty"`
	CurrentPassword        *string `json:"current_password,omitempty"` 
	NewPassword            *string `json:"new_password,omitempty"`     
	Gender                 *string `json:"gender,omitempty"`
	ProfilePictureURL      *string `json:"profile_picture_url,omitempty"`
	BannerURL              *string `json:"banner_url,omitempty"`         
	DateOfBirth            *string `json:"date_of_birth,omitempty"`      
	AccountPrivacy         *string `json:"account_privacy,omitempty"`    
	SubscribedToNewsletter *bool   `json:"subscribed_to_newsletter,omitempty"`
}


func (h *AuthHandler) HealthCheck(c *gin.Context) {
	resp, err := h.userClient.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": resp.Status})
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with name, username, email, password, gender, date of birth, security question and answer, and optional profile picture and banner URLs
// @Tags auth
// @Accept json
// @Produce json
// @Param        payload  body  RegisterPayload  true  "User registration details"
// @Success      200      {object}  emptypb.Empty		"Empty response on successful registration"
// @Failure      400      {object}  map[string]string	"Invalid request body"
// @Failure      409      {object}  map[string]string	"Conflict (e.g., email already exists)"
// @Failure      500      {object}  map[string]string	"Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var payload RegisterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

    // success, err := gwUtils.VerifyRecaptcha(payload.RecaptchaToken, c.ClientIP())
    // if err != nil || !success {
    //     errMsg := "reCAPTCHA verification failed"
    //     if err != nil { errMsg = err.Error() }
    //     c.JSON(http.StatusForbidden, gin.H{"error": errMsg})
    //     return
    // }

	// Prepare gRPC request (without reCAPTCHA token)
	grpcReq := &userpb.RegisterRequest{
		Name:             payload.Name,
		Username:         payload.Username,
		Email:            payload.Email,
		Password:         payload.Password,
		Gender:           payload.Gender,
		DateOfBirth:      payload.DateOfBirth,
		SecurityQuestion: payload.SecurityQuestion,
		SecurityAnswer:   payload.SecurityAnswer,
		SubscribedToNewsletter: payload.SubscribedToNewsletter,
	}

	if payload.ProfilePictureURL != nil {
		grpcReq.ProfilePictureUrl = payload.ProfilePictureURL
	}
	if payload.BannerURL != nil {
		grpcReq.BannerUrl = payload.BannerURL
	}
	// Forward request to User Service via gRPC client
	resp, err := h.userClient.Register(c.Request.Context(), grpcReq)
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
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// success, err := gwUtils.VerifyRecaptcha(payload.RecaptchaToken, c.ClientIP())
    // if err != nil || !success {
    //     errMsg := "reCAPTCHA verification failed"
    //     if err != nil { errMsg = err.Error() }
    //     c.JSON(http.StatusForbidden, gin.H{"error": errMsg})
    //     return
    // }

	grpcReq := &userpb.LoginRequest{
		Email:    payload.Email,
		Password: payload.Password,
	}

	resp, err := h.userClient.Login(c.Request.Context(), grpcReq)
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

func (h *AuthHandler) RefreshToken(c *gin.Context) {
    var req struct {
        RefreshToken string `json:"refresh_token"`
    }
    if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
        c.JSON(400, gin.H{"error": "refresh_token required"})
        return
    }

    // Validate the refresh token
    token, err := gwUtils.ValidateToken(req.RefreshToken)
    if err != nil || !token.Valid {
        c.JSON(401, gin.H{"error": "Invalid or expired refresh token"})
        return
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || claims["sub"] == nil {
        c.JSON(401, gin.H{"error": "Invalid token claims"})
        return
    }

    userID := uint(claims["sub"].(float64)) // adjust type as needed

    // Generate new tokens
    accessToken, refreshToken, err := gwUtils.GenerateTokens(userID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate tokens"})
        return
    }

    c.JSON(200, gin.H{
        "access_token": accessToken,
        "refresh_token": refreshToken,
    })
}

func (h *AuthHandler) GetSecurityQuestion(c *gin.Context) {
	var payload GetSecurityQuestionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	grpcReq := &userpb.GetSecurityQuestionRequest{Email: payload.Email}
	resp, err := h.userClient.GetSecurityQuestion(c.Request.Context(), grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			errorMsg := st.Message()
			if st.Code() == codes.NotFound || st.Code() == codes.PermissionDenied {
				errorMsg = "Could not retrieve security question for this email."
			}
			c.JSON(httpCode, gin.H{"error": errorMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get security question: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_question": resp.SecurityQuestion})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var payload ResetPasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	grpcReq := &userpb.ResetPasswordRequest{
		Email:          payload.Email,
		SecurityAnswer: payload.SecurityAnswer,
		NewPassword:    payload.NewPassword,
	}

	_, err := h.userClient.ResetPassword(c.Request.Context(), grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			c.JSON(httpCode, gin.H{"error": st.Message()}) // Return specific error message from backend
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Password reset failed: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now log in."})
}

func (h *AuthHandler) ResendVerificationCode(c *gin.Context) {
    var payload ResendVerificationPayload
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
        return
    }

    grpcReq := &userpb.ResendVerificationCodeRequest{Email: payload.Email}
    _, err := h.userClient.ResendVerificationCode(c.Request.Context(), grpcReq)
    if err != nil {
        st, ok := status.FromError(err)
        if ok {
            httpCode := grpcStatusCodeToHTTP(st.Code())
            c.JSON(httpCode, gin.H{"error": st.Message()})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resend verification code: " + err.Error()})
        }
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Verification code resent successfully. Please check your email."})
}

// GetProfile godoc
// @Summary Get current user's profile
// @Description Retrieve the authenticated user's profile details
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} userpb.UserProfileResponse "User profile details"
// @Failure 401 {object} map[string]string "Authentication context missing or unauthorized"
// @Failure 404 {object} map[string]string "User profile not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/me/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// Retrieve userID set by the AuthMiddleware
	userIDAny, exists := c.Get("userID")
	if !exists {
		log.Println("ERROR: userID not found in context after AuthMiddleware")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication context missing"})
		return
	}

	// Type assert userID (it was set as uint in middleware)
	userID, ok := userIDAny.(uint)
	if !ok || userID == 0 {
		log.Printf("ERROR: Invalid userID type or value in context: %v (%T)", userIDAny, userIDAny)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid authentication context"})
		return
	}

	// Prepare gRPC request
	reqUserID := uint32(userID)
	grpcReq := &userpb.GetUserProfileRequest{
		UserIdToView:    reqUserID,
		RequesterUserId: &reqUserID,
	}

	// Call User Service
	resp, err := h.userClient.GetUserProfile(c.Request.Context(), grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpCode := grpcStatusCodeToHTTP(st.Code())
			errorMsg := st.Message()
			if st.Code() == codes.NotFound {
				errorMsg = "User profile not found."
			}
			c.JSON(httpCode, gin.H{"error": errorMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateOwnUserProfile godoc
// @Summary Update current user's profile
// @Description Update the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Param profile body UpdateProfilePayload true "Updated profile details"
// @Security BearerAuth
// @Success 200 {object} object "Updated user profile in frontend format"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 401 {object} map[string]string "Authentication context missing or unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/me/profile [put]
func (h *AuthHandler) UpdateOwnUserProfile(c *gin.Context) {
    userID, ok := getUserIDFromContext(c)
    if !ok { return }

    var payload UpdateProfilePayload
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
        return
    }

    grpcReq := &userpb.UpdateUserProfileRequest{UserId: userID}

    if payload.Name != nil { grpcReq.Name = payload.Name }
    if payload.Bio != nil { grpcReq.Bio = payload.Bio }
    if payload.CurrentPassword != nil { grpcReq.CurrentPassword = payload.CurrentPassword }
    if payload.NewPassword != nil { grpcReq.NewPassword = payload.NewPassword }
    if payload.Gender != nil { grpcReq.Gender = payload.Gender }
    if payload.ProfilePictureURL != nil { grpcReq.ProfilePictureUrl = payload.ProfilePictureURL }
    if payload.BannerURL != nil { grpcReq.BannerUrl = payload.BannerURL }
    if payload.DateOfBirth != nil { grpcReq.DateOfBirth = payload.DateOfBirth }
    if payload.AccountPrivacy != nil { grpcReq.AccountPrivacy = payload.AccountPrivacy }
    if payload.SubscribedToNewsletter != nil { grpcReq.SubscribedToNewsletter = payload.SubscribedToNewsletter }


    updatedUserPb, err := h.userClient.UpdateUserProfile(c.Request.Context(), grpcReq)
    if err != nil {
        handleGRPCError(c, "update user profile", err)
        return
    }

    feUser := mapPbUserToFrontendUser(updatedUserPb)

    c.JSON(http.StatusOK, feUser)
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

func mapPbUserToFrontendUser(pbUser *userpb.User) interface{} {
    if pbUser == nil { return nil }
    return gin.H{
        "id":                       pbUser.GetId(),
        "name":                     pbUser.GetName(),
        "username":                 pbUser.GetUsername(),
        "email":                    pbUser.GetEmail(),
        "gender":                   pbUser.GetGender(),
        "profile_picture":          pbUser.GetProfilePicture(),
        "banner":                   pbUser.GetBanner(),
        "date_of_birth":            pbUser.GetDateOfBirth(),
        "account_status":           pbUser.GetAccountStatus(),
        "account_privacy":          pbUser.GetAccountPrivacy(),
        "subscribed_to_newsletter": pbUser.GetSubscribedToNewsletter(),
        "bio":                      pbUser.GetBio(),
        "created_at":               pbUser.GetCreatedAt().AsTime().Format(time.RFC3339),
    }
}