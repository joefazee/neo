package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/joefazee/neo/internal/validator"
	"github.com/joefazee/neo/models"
)

type MockCountryRepo struct {
	mock.Mock
}

func (m *MockCountryRepo) GetByCode(ctx context.Context, code string) (*models.Country, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Country), args.Error(1)
}

func (m *MockCountryRepo) GetAll(ctx context.Context) ([]models.Country, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Country), args.Error(1)
}

func (m *MockCountryRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Country, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Country), args.Error(1)
}

func (m *MockCountryRepo) GetActive(ctx context.Context) ([]models.Country, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Country), args.Error(1)
}

func (m *MockCountryRepo) Create(ctx context.Context, country *models.Country) error {
	return m.Called(ctx, country).Error(0)
}

func (m *MockCountryRepo) Update(ctx context.Context, country *models.Country) error {
	return m.Called(ctx, country).Error(0)
}

func (m *MockCountryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// Mock formatter package function
var mockFormatPhone func(phone, countryCode string) (string, error)

type DTOValidationTestSuite struct {
	suite.Suite
	countryRepo *MockCountryRepo
	sanitizer   *MockSanitizer
	ctx         context.Context
}

func (suite *DTOValidationTestSuite) SetupTest() {
	suite.countryRepo = &MockCountryRepo{}
	suite.sanitizer = &MockSanitizer{}
	suite.ctx = context.Background()

	mockFormatPhone = func(phone, _ string) (string, error) {
		return phone, nil
	}
}

func TestDTOValidation(t *testing.T) {
	suite.Run(t, new(DTOValidationTestSuite))
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_ValidInput() {
	isActive := true
	country := &models.Country{
		ID:       uuid.New(),
		Code:     "US",
		IsActive: &isActive,
	}

	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(country, nil)

	mockFormatPhone = func(_, _ string) (string, error) {
		return "+1234567890", nil
	}

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.True(result)
	suite.True(v.Valid())
	suite.Equal("john@example.com", req.Email)
	suite.Equal(country.ID, req.CountryID)
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_EmptyFirstName() {
	suite.sanitizer.On("StripHTML", "").Return("")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")

	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "first_name")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_EmptyLastName() {
	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "").Return("")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")

	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "last_name")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_InvalidEmail() {
	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "invalid-email").Return("invalid-email")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")

	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "invalid-email",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "email")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_EmptyCountryCode() {
	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")

	suite.countryRepo.On("GetByCode", suite.ctx, "").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "country_code")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_InvalidCountryCode() {
	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.countryRepo.On("GetByCode", suite.ctx, "XX").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "XX",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "country_code")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_InactiveCountry() {
	isActive := false
	country := &models.Country{
		ID:       uuid.New(),
		Code:     "XX",
		IsActive: &isActive,
	}

	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.countryRepo.On("GetByCode", suite.ctx, "XX").Return(country, nil)

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "XX",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "country_code")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_InvalidPhoneNumber() {
	isActive := true
	country := &models.Country{
		ID:       uuid.New(),
		Code:     "US",
		IsActive: &isActive,
	}

	suite.sanitizer.On("StripHTML", "John").Return("John")
	suite.sanitizer.On("StripHTML", "Doe").Return("Doe")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "invalid").Return("invalid")
	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(country, nil)

	mockFormatPhone = func(_, _ string) (string, error) {
		return "", errors.New("invalid phone")
	}

	req := &RegisterUserRequest{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "invalid",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "phone_number")
}

func (suite *DTOValidationTestSuite) TestRegisterUserRequest_NameTooShort() {
	suite.sanitizer.On("StripHTML", "J").Return("J")
	suite.sanitizer.On("StripHTML", "D").Return("D")
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")

	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(nil, errors.New("not found"))

	req := &RegisterUserRequest{
		FirstName:   "J",
		LastName:    "D",
		Email:       "john@example.com",
		CountryCode: "US",
		PhoneNumber: "+1234567890",
		Password:    "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "first_name")
	suite.Contains(v.Errors, "last_name")
}

func (suite *DTOValidationTestSuite) TestLoginRequest_ValidInput() {
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.True(result)
	suite.True(v.Valid())
}

func (suite *DTOValidationTestSuite) TestLoginRequest_EmptyIdentity() {
	suite.sanitizer.On("StripHTML", "").Return("")

	req := &LoginRequest{
		Identity: "",
		Password: "password123",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "identity")
}

func (suite *DTOValidationTestSuite) TestLoginRequest_EmptyPassword() {
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "password")
}

func (suite *DTOValidationTestSuite) TestLoginRequest_PasswordTooShort() {
	suite.sanitizer.On("StripHTML", "john@example.com").Return("john@example.com")

	req := &LoginRequest{
		Identity: "john@example.com",
		Password: "short",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "password")
}

func (suite *DTOValidationTestSuite) TestLoginRequest_WithValidCountryCode() {
	isActive := true
	country := &models.Country{
		ID:       uuid.New(),
		Code:     "US",
		IsActive: &isActive,
	}

	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.sanitizer.On("StripHTML", "US").Return("US")
	suite.countryRepo.On("GetByCode", suite.ctx, "US").Return(country, nil)

	mockFormatPhone = func(_, _ string) (string, error) {
		return "+1234567890", nil
	}

	req := &LoginRequest{
		Identity:    "+1234567890",
		Password:    "password123",
		CountryCode: "US",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.True(result)
	suite.True(v.Valid())
	suite.Equal("+1234567890", req.Identity)
}

func (suite *DTOValidationTestSuite) TestLoginRequest_WithInvalidCountryCode() {
	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.sanitizer.On("StripHTML", "XX").Return("XX")
	suite.countryRepo.On("GetByCode", suite.ctx, "XX").Return(nil, errors.New("not found"))

	req := &LoginRequest{
		Identity:    "+1234567890",
		Password:    "password123",
		CountryCode: "XX",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "country_code")
}

func (suite *DTOValidationTestSuite) TestLoginRequest_WithInactiveCountry() {
	isActive := false
	country := &models.Country{
		ID:       uuid.New(),
		Code:     "XX",
		IsActive: &isActive,
	}

	suite.sanitizer.On("StripHTML", "+1234567890").Return("+1234567890")
	suite.sanitizer.On("StripHTML", "XX").Return("XX")
	suite.countryRepo.On("GetByCode", suite.ctx, "XX").Return(country, nil)

	req := &LoginRequest{
		Identity:    "+1234567890",
		Password:    "password123",
		CountryCode: "XX",
	}

	v := validator.New()
	result := req.Validate(suite.ctx, v, suite.countryRepo, suite.sanitizer)

	suite.False(result)
	suite.False(v.Valid())
	suite.Contains(v.Errors, "country_code")
}
