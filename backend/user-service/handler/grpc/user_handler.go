package grpc

import (
	"context"
	"errors"
	"log"
	"regexp"
	"time"
	"unicode"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/repository/postgres"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/utils"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	nameRegex  = regexp.MustCompile(`^[a-zA-Z ]+$`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type UserHandler struct {
	userpb.UnimplementedUserServiceServer
	repo *postgres.UserRepository
}

func NewUserHandler(repo *postgres.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*userpb.HealthResponse, error) {
	log.Printf("Received HealthCheck request")
	if err := h.repo.CheckHealth(ctx); err != nil {
		return &userpb.HealthResponse{Status: "User Service is DEGRADED (DB Error)"}, nil
	}
	return &userpb.HealthResponse{Status: "User Service is OK (DB Connected)"}, nil
}

func validatePasswordComplexity(password string) error {
	var (
		hasMinLen  = len(password) >= 8
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		}
	}

	if !hasMinLen {
		return errors.New("password must be at least 8 characters long")
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}

	return nil
}

// Register implements user registration
func (h *UserHandler) Register(ctx context.Context, req *userpb.RegisterRequest) (*emptypb.Empty, error) {
	log.Printf("Received Register request for email: %s", req.Email)

	if req.Name == "" || req.Username == "" || req.Email == "" || req.Password == "" || req.SecurityQuestion == "" || req.SecurityAnswer == "" || req.DateOfBirth == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing required registration fields")
	}

	if len(req.Name) <= 4 {
		return nil, status.Errorf(codes.InvalidArgument, "Name must be longer than 4 characters")
	}
	if !nameRegex.MatchString(req.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "Name must contain only letters and spaces")
	}

	if !emailRegex.MatchString(req.Email) {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid email format")
	}

	if err := validatePasswordComplexity(req.Password); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if req.Gender != "" && req.Gender != "male" && req.Gender != "female" {
			return nil, status.Errorf(codes.InvalidArgument, "Gender must be 'male', 'female', or left empty")
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid date of birth format. Use YYYY-MM-DD.")
	}
	thirteenYearsAgo := time.Now().AddDate(-13, 0, 0)
	if dob.After(thirteenYearsAgo) {
		return nil, status.Errorf(codes.InvalidArgument, "You must be at least 13 years old to register")
	}

	// Generate Verification Code
	verificationCode, err := utils.GenerateVerificationCode(6)
	if err != nil {
		log.Printf("Error generating verification code: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to prepare registration")
	}

	// Create User struct for repository
	user := &postgres.User{
		Name:             req.Name,
		Username:         req.Username,
		Email:            req.Email,
		Gender:           req.Gender, // Optional, handle if empty if needed
		DateOfBirth:      dob.Format("2006-01-02"),
		SecurityQuestion: req.SecurityQuestion,
		// Passwords/Answers are hashed inside CreateUser
	}

	// Call repository to create user (handles hashing)
	err = h.repo.CreateUser(ctx, user, req.Password, req.SecurityAnswer, verificationCode)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		if err.Error() == "username or email already exists" {
			return nil, status.Errorf(codes.AlreadyExists, "Username or email already registered")
		}
		return nil, status.Errorf(codes.Internal, "Failed to register user: %v", err)
	}

	log.Printf("User pending verification: %d (%s)", user.ID, user.Email)

	go func() {
		errEmail := utils.SendVerificationEmail(user.Email, verificationCode)
		if errEmail != nil {
			log.Printf("Failed to send verification email to %s (user ID: %d): %v", user.Email, user.ID, errEmail)
		}
	}()

	// Return success (empty response)
	return &emptypb.Empty{}, nil
}

func (h *UserHandler) VerifyEmail(ctx context.Context, req *userpb.VerifyEmailRequest) (*emptypb.Empty, error) {
	log.Printf("Received VerifyEmail request for email: %s", req.Email)

	if req.Email == "" || req.Code == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Email and verification code are required")
	}

	// 1. Find user by email
	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("VerifyEmail failed for %s: %v", req.Email, err)
		if err.Error() == "user not found" {
			return nil, status.Errorf(codes.NotFound, "User not found or code is invalid")
		}
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user")
	}

	// 2. Check status and code
	if user.AccountStatus != "pending_verification" {
		log.Printf("VerifyEmail attempt for non-pending account: User %d, Status %s", user.ID, user.AccountStatus)
		// Could be already active or banned etc.
		return nil, status.Errorf(codes.FailedPrecondition, "Account is not awaiting verification")
	}

	if user.EmailVerificationCode == "" {
		log.Printf("VerifyEmail error: No verification code found for user %d", user.ID)
		return nil, status.Errorf(codes.Internal, "Verification code missing")
	}

	if user.EmailVerificationCode != req.Code {
		log.Printf("VerifyEmail failed: Invalid code for user %d", user.ID)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid verification code")
	}

	// 3. Activate Account
	err = h.repo.ActivateUserAccount(ctx, user.ID)
	if err != nil {
		log.Printf("Failed to activate account for user %d: %v", user.ID, err)
		return nil, status.Errorf(codes.Internal, "Failed to activate account")
	}

	log.Printf("Account successfully verified and activated for user: %d (%s)", user.ID, user.Email)

	// 4. Send Welcome Email
	go func(email, name string) {
		errWelcome := utils.SendWelcomeEmail(email, name)
		if errWelcome != nil {
			log.Printf("Failed to send welcome email to %s (user ID: %d): %v", email, user.ID, errWelcome)
		}
	}(user.Email, user.Name)

	return &emptypb.Empty{}, nil
}


// Login implements user login
func (h *UserHandler) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	log.Printf("Received Login request for email: %s", req.Email)

	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Email and password are required")
	}

	// Get user by email
	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("Login failed for %s: %v", req.Email, err)
		if err.Error() == "user not found" {
			return nil, status.Errorf(codes.NotFound, "Invalid email or password") // Generic message for security
		}
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user information")
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		log.Printf("Invalid password attempt for user %d (%s)", user.ID, user.Email)
		return nil, status.Errorf(codes.Unauthenticated, "Invalid email or password") // Generic message
	}

	// Check account status (e.g., 'active', 'banned', 'deactivated')
	if user.AccountStatus != "active" {
		log.Printf("Login attempt failed for inactive/banned user %d (%s), status: %s", user.ID, user.Email, user.AccountStatus)
		return nil, status.Errorf(codes.PermissionDenied, "Account is not active.")
	}

	log.Printf("User logged in successfully: %d (%s)", user.ID, user.Email)

	// Generate JWT Tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.ID)
	if err != nil {
		log.Printf("Error generating tokens for user %d after login: %v", user.ID, err)
		return nil, status.Errorf(codes.Internal, "Login successful, but failed to generate authentication tokens")
	}

	return &userpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *UserHandler) GetSecurityQuestion(ctx context.Context, req *userpb.GetSecurityQuestionRequest) (*userpb.GetSecurityQuestionResponse, error) {
	log.Printf("Received GetSecurityQuestion request for email: %s", req.Email)

	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Email is required")
	}

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("GetSecurityQuestion failed for %s: %v", req.Email, err)
		// Return NotFound even if email exists
		return nil, status.Errorf(codes.NotFound, "Email not found or account inactive")
	}

	// Ensure user account is active/verified before allowing password reset
	if user.AccountStatus != "active" {
		log.Printf("GetSecurityQuestion attempt for non-active user %d (%s), status: %s", user.ID, user.Email, user.AccountStatus)
		return nil, status.Errorf(codes.PermissionDenied, "Account status prevents password reset")
	}

	log.Printf("Returning security question for user %d", user.ID)
	return &userpb.GetSecurityQuestionResponse{
		SecurityQuestion: user.SecurityQuestion,
	}, nil
}

func (h *UserHandler) ResetPassword(ctx context.Context, req *userpb.ResetPasswordRequest) (*emptypb.Empty, error) {
	log.Printf("Received ResetPassword request for email: %s", req.Email)

	if req.Email == "" || req.SecurityAnswer == "" || req.NewPassword == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Email, security answer, and new password are required")
	}

	// 1. Get user by email
	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("ResetPassword failed for %s: %v", req.Email, err)
		return nil, status.Errorf(codes.NotFound, "Email not found or invalid credentials")
	}

    // 2. Check account status again
    if user.AccountStatus != "active" {
         log.Printf("ResetPassword attempt for non-active user %d (%s), status: %s", user.ID, user.Email, user.AccountStatus)
        return nil, status.Errorf(codes.PermissionDenied, "Account status prevents password reset")
    }

	// 3. Verify Security Answer
	err = bcrypt.CompareHashAndPassword([]byte(user.SecurityAnswerHash), []byte(req.SecurityAnswer))
	if err != nil {
		log.Printf("Invalid security answer attempt for user %d (%s)", user.ID, user.Email)
		return nil, status.Errorf(codes.Unauthenticated, "Invalid security answer")
	}
	log.Printf("Security answer verified for user %d", user.ID)

	// 4. Validate New Password (Complexity + ensure it's different)
	if err := validatePasswordComplexity(req.NewPassword); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "New password validation failed: %v", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.NewPassword))
	if err == nil {
		log.Printf("ResetPassword attempt failed for user %d: New password cannot be the same as the old one.", user.ID)
		return nil, status.Errorf(codes.InvalidArgument, "New password cannot be the same as the old password")
	} else if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
        log.Printf("ERROR comparing old/new password hash for user %d: %v", user.ID, err)
        return nil, status.Errorf(codes.Internal, "Failed to validate new password")
    }

	// 5. Hash the *new* password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR hashing new password for user %d: %v", user.ID, err)
		return nil, status.Errorf(codes.Internal, "Failed to secure new password")
	}

	// 6. Update password in repository
	err = h.repo.UpdatePassword(ctx, user.ID, string(newPasswordHash))
	if err != nil {
		log.Printf("Failed to update password in DB for user %d: %v", user.ID, err)
		return nil, status.Errorf(codes.Internal, "Failed to update password")
	}

	log.Printf("Password reset successfully for user %d (%s)", user.ID, user.Email)
	return &emptypb.Empty{}, nil
}