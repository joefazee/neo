package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/joefazee/neo/internal/security"
	"github.com/joefazee/neo/models"
)

type ServiceTestSuite struct {
	suite.Suite
	service    Service
	repo       *MockRepo
	tokenMaker *security.MockMaker
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.repo = &MockRepo{}
	suite.tokenMaker = &security.MockMaker{}
	suite.service = NewService(suite.repo, suite.tokenMaker)
}

func TestUserService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (suite *ServiceTestSuite) TestRegister_Success() {
	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		CountryID:   uuid.New(),
	}

	suite.repo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
		return user.Email == req.Email && user.FirstName == req.FirstName
	})).Run(func(args mock.Arguments) {
		user := args.Get(1).(*models.User)
		user.ID = uuid.New()
		user.CreatedAt = time.Now()
	}).Return(nil)

	result, err := suite.service.Register(context.Background(), req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(req.Email, result.Email)
	suite.Equal(req.FirstName, result.FirstName)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ServiceTestSuite) TestRegister_RepositoryError() {
	req := &RegisterUserRequest{
		Password: "password123",
	}

	suite.repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	result, err := suite.service.Register(context.Background(), req)

	suite.Error(err)
	suite.Nil(result)
}

func (suite *ServiceTestSuite) TestLogin_SuccessWithEmail() {
	user := &models.User{
		ID:           uuid.New(),
		Email:        "john@example.com",
		FirstName:    "John",
		LastName:     "Doe",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		UpdatedAt:    time.Now(),
	}

	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(user, nil)
	suite.tokenMaker.On("CreateToken", user.ID, 24*time.Hour, mock.AnythingOfType("int64"), security.TokenScopeAccess).Return("token123", &security.Payload{}, nil)

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("token123", result.AccessToken)
	suite.Equal(user.ID, result.User.ID)
}

func (suite *ServiceTestSuite) TestLogin_SuccessWithPhone() {
	user := &models.User{
		ID:           uuid.New(),
		Phone:        "+1234567890",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi",
		UpdatedAt:    time.Now(),
	}

	suite.repo.On("GetByPhone", mock.Anything, "+1234567890").Return(user, nil)
	suite.tokenMaker.On("CreateToken", user.ID, 24*time.Hour, mock.AnythingOfType("int64"), security.TokenScopeAccess).Return("token123", &security.Payload{}, nil)

	req := &LoginRequest{
		Identity: "+1234567890",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.NoError(err)
	suite.NotNil(result)
}

func (suite *ServiceTestSuite) TestLogin_UserNotFound() {
	suite.repo.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, gorm.ErrRecordNotFound)

	req := &LoginRequest{
		Identity: "notfound@example.com",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.Error(err)
	suite.Nil(result)
	suite.Equal("invalid credentials", err.Error())
}

func (suite *ServiceTestSuite) TestLogin_InvalidPassword() {
	user := &models.User{
		ID:           uuid.New(),
		Email:        "john@example.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi",
	}

	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(user, nil)

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "wrongpassword",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.Error(err)
	suite.Nil(result)
	suite.Equal("invalid credentials", err.Error())
}

func (suite *ServiceTestSuite) TestLogin_TokenCreationError() {
	user := &models.User{
		ID:           uuid.New(),
		Email:        "john@example.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi",
		UpdatedAt:    time.Now(),
	}

	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(user, nil)
	suite.tokenMaker.On("CreateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", nil, errors.New("token error"))

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.Error(err)
	suite.Nil(result)
}

func (suite *ServiceTestSuite) TestLogin_RepositoryError() {
	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, errors.New("db error"))

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.Error(err)
	suite.Nil(result)
}

func (suite *ServiceTestSuite) TestRequestPasswordReset_UserExists() {
	user := &models.User{ID: uuid.New()}
	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(user, nil)

	err := suite.service.RequestPasswordReset(context.Background(), "john@example.com")

	suite.NoError(err)
}

func (suite *ServiceTestSuite) TestRequestPasswordReset_UserNotFound() {
	suite.repo.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, gorm.ErrRecordNotFound)

	err := suite.service.RequestPasswordReset(context.Background(), "notfound@example.com")

	suite.NoError(err) // Should not reveal if user exists
}

func (suite *ServiceTestSuite) TestResetPassword() {
	err := suite.service.ResetPassword(context.Background(), "token", "newpassword")

	suite.NoError(err) // Currently always returns nil
}

func (suite *ServiceTestSuite) TestAssignRole_Success() {
	userID := uuid.New()
	roleID := uuid.New()

	suite.repo.On("AssignRole", mock.Anything, userID, roleID).Return(nil)

	err := suite.service.AssignRole(context.Background(), userID, roleID)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ServiceTestSuite) TestAssignRole_Error() {
	userID := uuid.New()
	roleID := uuid.New()

	suite.repo.On("AssignRole", mock.Anything, userID, roleID).Return(errors.New("assign error"))

	err := suite.service.AssignRole(context.Background(), userID, roleID)

	suite.Error(err)
}

func (suite *ServiceTestSuite) TestLogin_ZeroUpdatedAt() {
	user := &models.User{
		ID:           uuid.New(),
		Email:        "john@example.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi",
		UpdatedAt:    time.Time{},
	}

	suite.repo.On("GetByEmail", mock.Anything, "john@example.com").Return(user, nil)
	suite.tokenMaker.On("CreateToken", user.ID, 24*time.Hour, int64(0), security.TokenScopeAccess).Return("token123", &security.Payload{}, nil)

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "password",
	}

	result, err := suite.service.Login(context.Background(), req)

	suite.NoError(err)
	suite.NotNil(result)
}

func (suite *ServiceTestSuite) TestRegister_HashPasswordError() {
	req := &RegisterUserRequest{
		Password: "123",
	}
	result, err := suite.service.Register(context.Background(), req)
	suite.Error(err)
	suite.Nil(result)
}
