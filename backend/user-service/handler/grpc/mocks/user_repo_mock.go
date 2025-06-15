// backend/user-service/handler/grpc/mocks/user_repo_mock.go
package mocks

import (
	"context"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/repository/postgres"
	"github.com/stretchr/testify/mock"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) CheckHealth(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserRepo) CreateUser(ctx context.Context, user *postgres.User, plainPassword string, plainSecurityAnswer string, verificationCode string) error {
	args := m.Called(ctx, user, plainPassword, plainSecurityAnswer, verificationCode)
	return args.Error(0)
}

func (m *MockUserRepo) GetUserByEmail(ctx context.Context, email string) (*postgres.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*postgres.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByUsername(ctx context.Context, username string) (*postgres.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*postgres.User), args.Error(1)
}

func (m *MockUserRepo) ActivateUserAccount(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepo) UpdateVerificationCode(ctx context.Context, email string, newCode string) (*postgres.User, error) {
	args := m.Called(ctx, email, newCode)
	return args.Get(0).(*postgres.User), args.Error(1)
}

func (m *MockUserRepo) UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error {
	args := m.Called(ctx, userID, newPasswordHash)
	return args.Error(0)
}

func (m *MockUserRepo) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) (*postgres.User, error) {
	args := m.Called(ctx, userID, updates)
	return args.Get(0).(*postgres.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByID(ctx context.Context, userID uint) (*postgres.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*postgres.User), args.Error(1)
}

func (m *MockUserRepo) GetUsersByIDs(ctx context.Context, userIDs []uint) ([]postgres.User, error) {
	args := m.Called(ctx, userIDs)
	return args.Get(0).([]postgres.User), args.Error(1)
}

func (m *MockUserRepo) FollowUser(ctx context.Context, followerID, followedID uint) error {
	args := m.Called(ctx, followerID, followedID)
	return args.Error(0)
}

func (m *MockUserRepo) UnfollowUser(ctx context.Context, followerID, followedID uint) error {
	args := m.Called(ctx, followerID, followedID)
	return args.Error(0)
}

func (m *MockUserRepo) BlockUser(ctx context.Context, blockerID, blockedID uint) error {
	args := m.Called(ctx, blockerID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepo) UnblockUser(ctx context.Context, blockerID, blockedID uint) error {
	args := m.Called(ctx, blockerID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepo) GetFollowerCount(ctx context.Context, userID uint) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) GetFollowingCount(ctx context.Context, userID uint) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) GetFollowers(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockUserRepo) GetFollowing(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockUserRepo) IsFollowing(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	args := m.Called(ctx, requestUserID, targetUserID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepo) HasBlocked(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	args := m.Called(ctx, requestUserID, targetUserID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepo) IsBlockedBy(ctx context.Context, requestUserID, targetUserID uint) (bool, error) {
	args := m.Called(ctx, requestUserID, targetUserID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepo) GetBlockedUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockUserRepo) GetBlockingUserIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockUserRepo) GetFollowingIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]uint), args.Error(1)
}
