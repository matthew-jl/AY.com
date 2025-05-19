package http

import (
	"log"
	"net/http"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	// gwUtils "github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/utils"
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
	grpcReq := &userpb.GetUserProfileRequest{
		UserId: uint32(userID),
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