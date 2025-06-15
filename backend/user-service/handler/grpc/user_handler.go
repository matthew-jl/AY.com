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
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	nameRegex  = regexp.MustCompile(`^[a-zA-Z ]+$`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type UserHandler struct {
	userpb.UnimplementedUserServiceServer
	repo postgres.IUserRepo
}

// matches notification-service event/consumer.go
type NewFollowerEventPayload struct {
    FollowedUserID   uint32 `json:"followed_user_id"`
    FollowerUserID   uint32 `json:"follower_user_id"`
    FollowerUsername string `json:"follower_username"`
}

func NewUserHandler(repo postgres.IUserRepo) *UserHandler {
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
		Gender:           req.Gender,
		DateOfBirth:      dob.Format("2006-01-02"),
		SecurityQuestion: req.SecurityQuestion,
		SubscribedToNewsletter: req.SubscribedToNewsletter,
		ProfilePicture: req.GetProfilePictureUrl(),
		Banner: req.GetBannerUrl(),
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

func (h *UserHandler) ResendVerificationCode(ctx context.Context, req *userpb.ResendVerificationCodeRequest) (*emptypb.Empty, error) {
    log.Printf("Received ResendVerificationCode request for email: %s", req.Email)
    if req.Email == "" {
        return nil, status.Errorf(codes.InvalidArgument, "Email is required")
    }

    newVerificationCode, err := utils.GenerateVerificationCode(6)
    if err != nil {
        log.Printf("Error generating new verification code for resend: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to prepare for resend")
    }

    user, err := h.repo.UpdateVerificationCode(ctx, req.Email, newVerificationCode)
    if err != nil {
        log.Printf("Error updating verification code for user %s: %v", req.Email, err)
        if err.Error() == "user not found with this email" {
            return nil, status.Errorf(codes.NotFound, "%s", err.Error())
        }
        if err.Error() == "account is already active" {
            return nil, status.Errorf(codes.FailedPrecondition, "%s", err.Error())
        }
        return nil, status.Errorf(codes.Internal, "Failed to update verification details")
    }

    go func(email, code string) {
        errEmail := utils.SendVerificationEmail(email, code)
        if errEmail != nil {
            log.Printf("Failed to resend verification email to %s (user ID: %d): %v", email, user.ID, errEmail)
        }
    }(user.Email, newVerificationCode)

    log.Printf("New verification code sent to %s", req.Email)
    return &emptypb.Empty{}, nil
}


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
			return nil, status.Errorf(codes.NotFound, "Invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user information")
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		log.Printf("Invalid password attempt for user %d (%s)", user.ID, user.Email)
		return nil, status.Errorf(codes.Unauthenticated, "Invalid email or password")
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

	// 5. Hash the new password
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

func (h *UserHandler) UpdateUserProfile(ctx context.Context, req *userpb.UpdateUserProfileRequest) (*userpb.User, error) {
	userID := uint(req.GetUserId())
	log.Printf("Received UpdateUserProfile request for User ID: %d", userID)

	if userID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required for update")
	}

	currentUser, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("UpdateUserProfile: User %d not found: %v", userID, err)
		if err.Error() == "user not found by ID" {
			return nil, status.Errorf(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user for update")
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		if len(req.GetName()) <= 4 { return nil, status.Errorf(codes.InvalidArgument, "Name must be longer than 4 characters") }
		if !nameRegex.MatchString(req.GetName()) { return nil, status.Errorf(codes.InvalidArgument, "Name must contain only letters and spaces")}
		updates["name"] = req.GetName()
	}
	// if req.Username != nil { updates["username"] = req.GetUsername() }
	// if req.Email != nil { updates["email"] = req.GetEmail() }

	if req.NewPassword != nil && req.GetNewPassword() != "" {
		if req.CurrentPassword == nil || req.GetCurrentPassword() == "" {
			return nil, status.Errorf(codes.InvalidArgument, "Current password is required to set a new password")
		}
		err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(req.GetCurrentPassword()))
		if err != nil {
			log.Printf("UpdateUserProfile: Invalid current password for user %d", userID)
			return nil, status.Errorf(codes.Unauthenticated, "Incorrect current password")
		}
		if err := validatePasswordComplexity(req.GetNewPassword()); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "New password validation failed: %v", err)
		}
		if bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(req.GetNewPassword())) == nil {
			return nil, status.Errorf(codes.InvalidArgument, "New password cannot be the same as the old password")
		}
		newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.GetNewPassword()), bcrypt.DefaultCost)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to hash new password")
		}
		updates["password_hash"] = string(newPasswordHash)
	}

	if req.Gender != nil {
		if g := req.GetGender(); g != "" && g != "male" && g != "female" {
			return nil, status.Errorf(codes.InvalidArgument, "Gender must be 'male', 'female', or empty")
		}
		updates["gender"] = req.GetGender()
	}
	if req.ProfilePictureUrl != nil { updates["profile_picture"] = req.GetProfilePictureUrl() }
	if req.BannerUrl != nil { updates["banner"] = req.GetBannerUrl() }
	if req.DateOfBirth != nil && req.GetDateOfBirth() != "" {
		dob, err := time.Parse("2006-01-02", req.GetDateOfBirth())
		if err != nil { return nil, status.Errorf(codes.InvalidArgument, "Invalid date of birth format") }
        thirteenYearsAgo := time.Now().AddDate(-13, 0, 0)
        if dob.After(thirteenYearsAgo) { return nil, status.Errorf(codes.InvalidArgument, "You must be at least 13 years old") }
		updates["date_of_birth"] = dob
	}
	if req.Bio != nil { updates["bio"] = req.GetBio() }
	if req.AccountPrivacy != nil {
        privacy := req.GetAccountPrivacy()
        if privacy != "public" && privacy != "private" {
             return nil, status.Errorf(codes.InvalidArgument, "Account privacy must be 'public' or 'private'")
        }
		updates["account_privacy"] = privacy
	}
	if req.SubscribedToNewsletter != nil { updates["subscribed_to_newsletter"] = req.GetSubscribedToNewsletter() }


	if len(updates) == 0 {
		log.Printf("UpdateUserProfile: No update fields provided for user %d", userID)
		return mapDBUserToProtoUser(currentUser), nil
	}

	updatedUser, err := h.repo.UpdateUser(ctx, userID, updates)
	if err != nil {
		log.Printf("UpdateUserProfile: Failed to update user %d: %v", userID, err)
		return nil, status.Errorf(codes.Internal, "Failed to update profile")
	}

	log.Printf("User profile updated successfully for User ID: %d", userID)
	return mapDBUserToProtoUser(updatedUser), nil
}


func (h *UserHandler) GetUserProfile(ctx context.Context, req *userpb.GetUserProfileRequest) (*userpb.UserProfileResponse, error) {
	log.Printf("Received GetUserProfile request for UserToView: %d, Requester: %d", req.UserIdToView, req.GetRequesterUserId())
	if req.UserIdToView == 0 { return nil, status.Errorf(codes.InvalidArgument, "User ID to view is required") }

	targetUser, err := h.repo.GetUserByID(ctx, uint(req.UserIdToView))
	if err != nil {
		log.Printf("GetUserProfile failed for user ID %d: %v", req.UserIdToView, err)
	}

	// Privacy Check
	requesterID := uint(req.GetRequesterUserId())
	isOwner := requesterID != 0 && requesterID == targetUser.ID

	isBlockedByTarget, _ := h.repo.IsBlockedBy(ctx, requesterID, targetUser.ID)
	if isBlockedByTarget && !isOwner {
		log.Printf("User %d blocked from viewing profile of user %d", requesterID, targetUser.ID)
		return nil, status.Errorf(codes.PermissionDenied, "You are blocked by this user.")
	}
	hasRequesterBlockedTarget, _ := h.repo.HasBlocked(ctx, requesterID, targetUser.ID)


	if targetUser.AccountPrivacy == "private" && !isOwner {
		isFollowing, _ := h.repo.IsFollowing(ctx, requesterID, targetUser.ID)
		if !isFollowing {
			log.Printf("Access denied to private profile of user %d for requester %d", targetUser.ID, requesterID)
			// return nil, status.Errorf(codes.PermissionDenied, "This account is private.")
		}
	}

	followerCount, _ := h.repo.GetFollowerCount(ctx, targetUser.ID)
	followingCount, _ := h.repo.GetFollowingCount(ctx, targetUser.ID)

	// Check relationship status if requester ID is provided
	var isFollowedByReq, isBlockedByReq bool
	if requesterID != 0 && requesterID != targetUser.ID {
		isFollowedByReq, _ = h.repo.IsFollowing(ctx, requesterID, targetUser.ID)
		isBlockedByReq = hasRequesterBlockedTarget
	}


	userProto := &userpb.User{
		Id:             uint32(targetUser.ID),
		Name:           targetUser.Name,
		Username:       targetUser.Username,
		Email:          targetUser.Email,
		Gender:         targetUser.Gender,
		ProfilePicture: targetUser.ProfilePicture,
		Banner:         targetUser.Banner,
		DateOfBirth:    targetUser.DateOfBirth,
		AccountStatus:  targetUser.AccountStatus,
		AccountPrivacy: targetUser.AccountPrivacy,
		SubscribedToNewsletter: targetUser.SubscribedToNewsletter,
		CreatedAt:      timestamppb.New(targetUser.CreatedAt),
		Bio:            targetUser.Bio,
	}

	return &userpb.UserProfileResponse{
		User:                   userProto,
		FollowerCount:          int32(followerCount),
		FollowingCount:         int32(followingCount),
		IsFollowedByRequester:  isFollowedByReq,
		IsBlockedByRequester:   isBlockedByReq,
        IsBlockingRequester:    isBlockedByTarget,
	}, nil
}

func (h *UserHandler) GetUserProfilesByIds(ctx context.Context, req *userpb.GetUserProfilesByIdsRequest) (*userpb.GetUserProfilesByIdsResponse, error) {
	log.Printf("Received GetUserProfilesByIds request for %d IDs", len(req.UserIds))
	if len(req.UserIds) == 0 {
		return &userpb.GetUserProfilesByIdsResponse{Users: make(map[uint32]*userpb.User)}, nil // Return empty map
	}

	// Convert uint32 IDs to uint for repository
	uintUserIDs := make([]uint, len(req.UserIds))
	for i, id := range req.UserIds {
		uintUserIDs[i] = uint(id)
	}

	dbUsers, err := h.repo.GetUsersByIDs(ctx, uintUserIDs)
	if err != nil {
		log.Printf("Error getting users by IDs: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user profiles")
	}

	userMap := make(map[uint32]*userpb.User)
	for _, dbUser := range dbUsers {
		userMap[uint32(dbUser.ID)] = &userpb.User{
			Id:             uint32(dbUser.ID),
			Name:           dbUser.Name,
			Username:       dbUser.Username,
			Email:          dbUser.Email,
			ProfilePicture: dbUser.ProfilePicture,
		}
	}
	return &userpb.GetUserProfilesByIdsResponse{Users: userMap}, nil
}

func (h *UserHandler) GetUserByUsername(ctx context.Context, req *userpb.GetUserByUsernameRequest) (*userpb.User, error) {
    log.Printf("Received GetUserByUsername request for Username: %s", req.Username)
    if req.Username == "" {
        return nil, status.Errorf(codes.InvalidArgument, "Username is required")
    }

    user, err := h.repo.GetUserByUsername(ctx, req.Username)
    if err != nil {
        log.Printf("GetUserByUsername failed for %s: %v", req.Username, err)
        if err.Error() == "user not found by username" {
            return nil, status.Errorf(codes.NotFound, "User not found")
        }
        return nil, status.Errorf(codes.Internal, "Failed to retrieve user by username")
    }

    return &userpb.User{
        Id:             uint32(user.ID),
        Name:           user.Name,
        Username:       user.Username,
        Email:          user.Email,
        ProfilePicture: user.ProfilePicture,
        Banner:         user.Banner,
        Bio:            user.Bio,
        AccountPrivacy: user.AccountPrivacy,
        CreatedAt:      timestamppb.New(user.CreatedAt),
    }, nil
}

func (h *UserHandler) FollowUser(ctx context.Context, req *userpb.FollowRequest) (*emptypb.Empty, error) {
	log.Printf("User %d attempts to follow user %d", req.FollowerId, req.FollowedId)
	if req.FollowerId == 0 || req.FollowedId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Follower and Followed IDs are required")
	}
	if req.FollowerId == req.FollowedId {
		return nil, status.Errorf(codes.InvalidArgument, "User cannot follow themselves")
	}
	// TODO: Check if Follower has blocked Followed, or if Followed has blocked Follower
	
	// Get follower profile to publish event
	followerProfile, err := h.GetUserProfile(ctx, &userpb.GetUserProfileRequest{
        UserIdToView: req.FollowerId,
    })
    if err != nil || followerProfile == nil || followerProfile.User == nil {
        log.Printf("FollowUser: Could not get profile for follower %d to publish event: %v", req.FollowerId, err)
    }
    followerUsername := "Someone" // Default
    if followerProfile != nil && followerProfile.User != nil {
        followerUsername = followerProfile.User.Username
    }

	err = h.repo.FollowUser(ctx, uint(req.FollowerId), uint(req.FollowedId))
	if err != nil {
		log.Printf("Error following user: %v", err)
		if err.Error() == "user cannot follow themselves" {return nil, status.Errorf(codes.InvalidArgument, err.Error())}
		return nil, status.Errorf(codes.Internal, "Could not process follow request")
	}

	// Publish Follow Event
	eventPayload := NewFollowerEventPayload{
		FollowedUserID:   req.FollowedId,
		FollowerUserID:   req.FollowerId,
        FollowerUsername: followerUsername,
	}
	// Use "social.new_follower" as routing key, "social_events" as exchange
	// These consts should match what notification-service consumer expects
	go func() {
		errPub := utils.PublishEvent(context.Background(), "social_events", "social.new_follower", eventPayload)
		if errPub != nil {
			log.Printf("ERROR publishing NewFollowerEvent for %d -> %d: %v", req.FollowerId, req.FollowedId, errPub)
		}
	}()


	return &emptypb.Empty{}, nil
}

func (h *UserHandler) UnfollowUser(ctx context.Context, req *userpb.FollowRequest) (*emptypb.Empty, error) {
	log.Printf("User %d attempts to unfollow user %d", req.FollowerId, req.FollowedId)
    if req.FollowerId == 0 || req.FollowedId == 0 { return nil, status.Errorf(codes.InvalidArgument, "IDs required")}
	err := h.repo.UnfollowUser(ctx, uint(req.FollowerId), uint(req.FollowedId))
	if err != nil {
		log.Printf("Error unfollowing user: %v", err)
		return nil, status.Errorf(codes.Internal, "Could not process unfollow request")
	}
	return &emptypb.Empty{}, nil
}

func (h *UserHandler) BlockUser(ctx context.Context, req *userpb.BlockRequest) (*emptypb.Empty, error) {
	log.Printf("User %d attempts to block user %d", req.BlockerId, req.BlockedId)
    if req.BlockerId == 0 || req.BlockedId == 0 { return nil, status.Errorf(codes.InvalidArgument, "IDs required")}
    if req.BlockerId == req.BlockedId { return nil, status.Errorf(codes.InvalidArgument, "User cannot block themselves")}
	// TODO: Transaction: remove existing follows (both ways) when blocking.
	err := h.repo.BlockUser(ctx, uint(req.BlockerId), uint(req.BlockedId))
	if err != nil {
		log.Printf("Error blocking user: %v", err)
		return nil, status.Errorf(codes.Internal, "Could not process block request")
	}
	return &emptypb.Empty{}, nil
}

func (h *UserHandler) UnblockUser(ctx context.Context, req *userpb.BlockRequest) (*emptypb.Empty, error) {
	log.Printf("User %d attempts to unblock user %d", req.BlockerId, req.BlockedId)
    if req.BlockerId == 0 || req.BlockedId == 0 { return nil, status.Errorf(codes.InvalidArgument, "IDs required")}
	err := h.repo.UnblockUser(ctx, uint(req.BlockerId), uint(req.BlockedId))
	if err != nil {
		log.Printf("Error unblocking user: %v", err)
		return nil, status.Errorf(codes.Internal, "Could not process unblock request")
	}
	return &emptypb.Empty{}, nil
}

func (h *UserHandler) GetFollowers(ctx context.Context, req *userpb.GetSocialListRequest) (*userpb.GetSocialListResponse, error) {
	log.Printf("GetFollowers for UserID: %d, Requester: %d, Page: %d", req.UserId, req.GetRequesterUserId(), req.Page)
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Target UserID is required")}
	limit, offset := getLimitOffset(req.Page, req.Limit)

	followerIDs, err := h.repo.GetFollowers(ctx, uint(req.UserId), limit, offset)
	if err != nil {
		log.Printf("Error retrieving followers: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve followers")
	}

	return h.hydrateSocialList(ctx, followerIDs, req.GetRequesterUserId(), len(followerIDs) == limit)
}

func (h *UserHandler) GetFollowing(ctx context.Context, req *userpb.GetSocialListRequest) (*userpb.GetSocialListResponse, error) {
	log.Printf("GetFollowing for UserID: %d, Requester: %d, Page: %d", req.UserId, req.GetRequesterUserId(), req.Page)
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Target UserID is required")}
	limit, offset := getLimitOffset(req.Page, req.Limit)

	followingIDs, err := h.repo.GetFollowing(ctx, uint(req.UserId), limit, offset)
	if err != nil {
		log.Printf("Error retrieving following: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve following")
	}

	return h.hydrateSocialList(ctx, followingIDs, req.GetRequesterUserId(), len(followingIDs) == limit)
}

func (h *UserHandler) GetBlockedUserIDs(ctx context.Context, req *userpb.SocialListRequest) (*userpb.UserIDListResponse, error) {
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "User ID is required") }
    limit, offset := getLimitOffset(req.Page, req.Limit) // Use your existing helper
    ids, err := h.repo.GetBlockedUserIDs(ctx, uint(req.UserId), limit, offset)
    if err != nil { return nil, status.Errorf(codes.Internal, "Failed to get blocked users: %v", err) }
    return &userpb.UserIDListResponse{UserIds: uintSliceToUint32Slice(ids), HasMore: len(ids) == limit}, nil
}

func (h *UserHandler) GetBlockingUserIDs(ctx context.Context, req *userpb.SocialListRequest) (*userpb.UserIDListResponse, error) {
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "User ID is required") }
    limit, offset := getLimitOffset(req.Page, req.Limit)
    ids, err := h.repo.GetBlockingUserIDs(ctx, uint(req.UserId), limit, offset)
    if err != nil { return nil, status.Errorf(codes.Internal, "Failed to get blocking users: %v", err) }
    return &userpb.UserIDListResponse{UserIds: uintSliceToUint32Slice(ids), HasMore: len(ids) == limit}, nil
}

func (h *UserHandler) GetFollowingIDs(ctx context.Context, req *userpb.SocialListRequest) (*userpb.UserIDListResponse, error) {
    if req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "User ID is required") }
    limit, offset := getLimitOffset(req.Page, req.Limit)
    ids, err := h.repo.GetFollowingIDs(ctx, uint(req.UserId), limit, offset)
    if err != nil { return nil, status.Errorf(codes.Internal, "Failed to get following list: %v", err)}
    return &userpb.UserIDListResponse{UserIds: uintSliceToUint32Slice(ids), HasMore: len(ids) == limit}, nil
}

func (h *UserHandler) HasBlocked(ctx context.Context, req *userpb.BlockCheckRequest) (*userpb.BlockStatusResponse, error) {
	log.Printf("Received HasBlocked request: Actor %d, Subject %d", req.ActorId, req.SubjectId)
	if req.ActorId == 0 || req.SubjectId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Actor ID and Subject ID are required")
	}
    if req.ActorId == req.SubjectId {
        return &userpb.BlockStatusResponse{IsTrue: false}, nil
    }

	hasBlocked, err := h.repo.HasBlocked(ctx, uint(req.ActorId), uint(req.SubjectId))
	if err != nil {
		log.Printf("Error checking HasBlocked in repository: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to check block status")
	}
	return &userpb.BlockStatusResponse{IsTrue: hasBlocked}, nil
}

func (h *UserHandler) IsBlockedBy(ctx context.Context, req *userpb.BlockCheckRequest) (*userpb.BlockStatusResponse, error) {
	log.Printf("Received IsBlockedBy request: Actor %d, Subject %d", req.ActorId, req.SubjectId)
	if req.ActorId == 0 || req.SubjectId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Actor ID and Subject ID are required")
	}
    if req.ActorId == req.SubjectId {
        return &userpb.BlockStatusResponse{IsTrue: false}, nil
    }

	isBlockedBy, err := h.repo.IsBlockedBy(ctx, uint(req.ActorId), uint(req.SubjectId))
	if err != nil {
		log.Printf("Error checking IsBlockedBy in repository: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to check block status")
	}
	return &userpb.BlockStatusResponse{IsTrue: isBlockedBy}, nil
}

func (h *UserHandler) IsFollowing(ctx context.Context, req *userpb.FollowCheckRequest) (*userpb.BlockStatusResponse, error) {
	log.Printf("Received IsFollowing request: Follower %d, Followed %d", req.FollowerId, req.FollowedId)
	if req.FollowerId == 0 || req.FollowedId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Follower ID and Followed ID are required")
	}
	if req.FollowerId == req.FollowedId {
		return &userpb.BlockStatusResponse{IsTrue: false}, nil
	}

	isFollowing, err := h.repo.IsFollowing(ctx, uint(req.FollowerId), uint(req.FollowedId))
	if err != nil {
		log.Printf("Error checking IsFollowing in repository: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to check follow status")
	}
	return &userpb.BlockStatusResponse{IsTrue: isFollowing}, nil
}

// --- Helpers ---
// Helper for pagination
func getLimitOffset(page, limit int32) (int, int) {
	p := int(page); l := int(limit)
	if l <= 0 || l > 50 { l = 20 }
	if p <= 0 { p = 1 }
	return l, (p - 1) * l
}

// Helper to hydrate a list of user IDs into SocialUser protos
func (h *UserHandler) hydrateSocialList(ctx context.Context, userIDs []uint, requesterID uint32, hasMore bool) (*userpb.GetSocialListResponse, error) {
	if len(userIDs) == 0 {
		return &userpb.GetSocialListResponse{Users: []*userpb.SocialUser{}, HasMore: false}, nil
	}

	protoUserIDs := make([]uint32, len(userIDs))
	for i, id := range userIDs { protoUserIDs[i] = uint32(id) }

	profilesResp, err := h.GetUserProfilesByIds(ctx, &userpb.GetUserProfilesByIdsRequest{UserIds: protoUserIDs})
	if err != nil {
		log.Printf("Error hydrating social list (GetUserProfilesByIds): %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user details for list")
	}

	socialUsers := make([]*userpb.SocialUser, 0, len(userIDs))
	for _, userID := range userIDs {
        profile, ok := profilesResp.Users[uint32(userID)]
        if !ok || profile == nil {
            log.Printf("Warning: Profile not found for userID %d during social list hydration", userID)
            continue
        }

		isFollowedByReq := false
		if requesterID != 0 && requesterID != uint32(userID) {
			isFollowedByReq, _ = h.repo.IsFollowing(ctx, uint(requesterID), userID)
		}
		socialUsers = append(socialUsers, &userpb.SocialUser{
			UserSummary:           profile,
			IsFollowedByRequester: isFollowedByReq,
		})
	}

	return &userpb.GetSocialListResponse{Users: socialUsers, HasMore: hasMore}, nil
}

// Helper to map postgres.User to userpb.User
func mapDBUserToProtoUser(dbUser *postgres.User) *userpb.User {
    if dbUser == nil { return nil }
    return &userpb.User{
        Id:             uint32(dbUser.ID),
        Name:           dbUser.Name,
        Username:       dbUser.Username,
        Email:          dbUser.Email,
        Gender:         dbUser.Gender,
        ProfilePicture: dbUser.ProfilePicture,
        Banner:         dbUser.Banner,
		DateOfBirth:    dbUser.DateOfBirth,
        AccountStatus:  dbUser.AccountStatus,
        AccountPrivacy: dbUser.AccountPrivacy,
        SubscribedToNewsletter: dbUser.SubscribedToNewsletter,
        Bio:            dbUser.Bio,
        CreatedAt:      timestamppb.New(dbUser.CreatedAt),
    }
}

func uintSliceToUint32Slice(u []uint) []uint32 {
    u32 := make([]uint32, len(u))
    for i, v := range u { u32[i] = uint32(v) }
    return u32
}