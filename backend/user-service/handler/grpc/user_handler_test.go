package grpc_test // Use _test package to avoid import cycles if IUserRepo is in same package

import (
	"context"
	"errors"
	"testing"

	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"    // Import the generated protobuf package
	userhandler "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/handler/grpc" // Import actual handler
	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/handler/grpc/mocks"       // Import the mock package
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestUserHandler_HealthCheck_OK(t *testing.T) {
	// 1. Arrange: Create mock repository and the handler with the mock
	mockRepo := new(mocks.MockUserRepo)
	// mediaClient can be nil if HealthCheck doesn't use it.
	// If it does, you'd need a mockMediaClient as well.
	handler := userhandler.NewUserHandler(mockRepo)

	// 2. Expectation: Tell the mock what to expect and what to return
	mockRepo.On("CheckHealth", mock.Anything).Return(nil).Once()
	// mock.AnythingOfType("*context.emptyCtx") or just mock.Anything for simplicity

	// 3. Act: Call the method being tested
	req := &emptypb.Empty{}
	resp, err := handler.HealthCheck(context.Background(), req)

	// 4. Assert: Check the results
	assert.NoError(t, err, "HealthCheck should not return an error on success")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Equal(t, "User Service is OK (DB Connected)", resp.Status, "Status message mismatch")

	// Verify that all expectations on the mock were met
	mockRepo.AssertExpectations(t)
}

func TestUserHandler_HealthCheck_DBError(t *testing.T) {
	// 1. Arrange
	mockRepo := new(mocks.MockUserRepo)
	handler := userhandler.NewUserHandler(mockRepo)

	// 2. Expectation
	dbError := errors.New("simulated database connection error")
	mockRepo.On("CheckHealth", mock.Anything).Return(dbError).Once()

	// 3. Act
	req := &emptypb.Empty{}
	resp, err := handler.HealthCheck(context.Background(), req)

	// 4. Assert
	assert.NoError(t, err, "HealthCheck itself should still not error out for this specific degraded case")
	assert.NotNil(t, resp)
	assert.Equal(t, "User Service is DEGRADED (DB Error)", resp.Status)
	mockRepo.AssertExpectations(t)
}

func TestUserHandler_Register_MissingRequiredFields(t *testing.T) {
	// 1. Arrange
	mockRepo := new(mocks.MockUserRepo)
	handler := userhandler.NewUserHandler(mockRepo)

	testCases := []struct {
		name    string
		req     *userpb.RegisterRequest
		wantErrMsgContains string
	}{
		{
			name: "missing name",
			req: &userpb.RegisterRequest{
				// Name: "", // Missing
				Username:         "testuser",
				Email:            "test@example.com",
				Password:         "Password123!",
				SecurityQuestion: "pet",
				SecurityAnswer:   "fluffy",
				DateOfBirth:      "2000-01-01",
			},
			wantErrMsgContains: "Missing required registration fields", // Or more specific if your validation changes
		},
		{
			name: "missing email",
			req: &userpb.RegisterRequest{
				Name:             "Test User",
				Username:         "testuser",
				// Email: "", // Missing
				Password:         "Password123!",
				SecurityQuestion: "pet",
				SecurityAnswer:   "fluffy",
				DateOfBirth:      "2000-01-01",
			},
			wantErrMsgContains: "Missing required registration fields",
		},
        // Add more cases for other missing required fields if your initial check is granular
        // For now, the handler has a single check:
        // if req.Name == "" || req.Username == "" || req.Email == "" || req.Password == "" || req.SecurityQuestion == "" || req.SecurityAnswer == "" || req.DateOfBirth == ""
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 2. Act
			_, err := handler.Register(context.Background(), tc.req)

			// 3. Assert
			assert.Error(t, err, "Expected an error for missing fields")
			st, ok := status.FromError(err)
			assert.True(t, ok, "Error should be a gRPC status error")
			assert.Equal(t, codes.InvalidArgument, st.Code(), "Expected InvalidArgument error code")
			assert.Contains(t, st.Message(), tc.wantErrMsgContains, "Error message mismatch")

			// Ensure no repository or client calls were made due to early validation failure
			mockRepo.AssertNotCalled(t, "CreateUser")
            mockRepo.AssertNotCalled(t, "GetUserByEmail") // If you check for existing email before this basic validation
		})
	}
}