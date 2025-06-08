package countries

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/joefazee/neo/tests/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestService_GetAllCountries(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		isActive := true
		countries := []models.Country{
			{ID: uuid.New(), Name: "Nigeria", Code: "NGA", IsActive: &isActive},
			{ID: uuid.New(), Name: "USA", Code: "USA", IsActive: &isActive},
		}

		mockRepo.On("GetAll", ctx).Return(countries, nil)

		result, err := srvc.GetAllCountries(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		mockRepo.On("GetAll", ctx).Return([]models.Country{}, assert.AnError)

		result, err := srvc.GetAllCountries(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetCountryByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		isActive := true
		country := &models.Country{
			ID:       id,
			Name:     "Nigeria",
			Code:     "NGA",
			IsActive: &isActive,
		}

		mockRepo.On("GetByID", ctx, id).Return(country, nil)

		result, err := srvc.GetCountryByID(ctx, id)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		mockRepo.On("GetByID", ctx, id).Return(nil, gorm.ErrRecordNotFound)

		result, err := srvc.GetCountryByID(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, models.ErrRecordNotFound, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		mockRepo.On("GetByID", ctx, id).Return(nil, assert.AnError)

		result, err := srvc.GetCountryByID(ctx, id)

		assert.Error(t, err)
		assert.NotEqual(t, models.ErrRecordNotFound, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetCountryByCode(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		isActive := true
		country := &models.Country{
			ID:       uuid.New(),
			Name:     "Nigeria",
			Code:     "NGA",
			IsActive: &isActive,
		}

		mockRepo.On("GetByCode", ctx, "NGA").Return(country, nil)

		result, err := srvc.GetCountryByCode(ctx, "nga") // lowercase input

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		mockRepo.On("GetByCode", ctx, "XYZ").Return(nil, gorm.ErrRecordNotFound)

		result, err := srvc.GetCountryByCode(ctx, "xyz")

		assert.Error(t, err)
		assert.Equal(t, models.ErrRecordNotFound, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetActiveCountries(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		isActive := true
		countries := []models.Country{
			{ID: uuid.New(), Name: "Nigeria", Code: "NGA", IsActive: &isActive},
		}

		mockRepo.On("GetActive", ctx).Return(countries, nil)

		result, err := srvc.GetActiveCountries(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_CreateCountry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		req := &CreateCountryRequest{
			Name:           "Nigeria",
			Code:           "nga",
			CurrencyCode:   "ngn",
			CurrencySymbol: "₦",
			ContractUnit:   decimal.NewFromInt(100),
			MinBet:         decimal.NewFromInt(10),
			MaxBet:         decimal.NewFromInt(1000),
			KYCRequired:    true,
			Timezone:       "Africa/Lagos",
		}

		mockRepo.On("GetByCode", ctx, "nga").Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Country")).Return(nil)

		result, err := srvc.CreateCountry(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Min/Max Bet", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		req := &CreateCountryRequest{
			Name:         "Nigeria",
			Code:         "NGA",
			CurrencyCode: "NGN",
			MinBet:       decimal.NewFromInt(1000),
			MaxBet:       decimal.NewFromInt(100), // max < min
		}

		result, err := srvc.CreateCountry(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "min_bet must be less than max_bet")
		assert.Nil(t, result)
	})

	t.Run("Country Code Already Exists", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		req := &CreateCountryRequest{
			Name:         "Nigeria",
			Code:         "NGA",
			CurrencyCode: "NGN",
			MinBet:       decimal.NewFromInt(10),
			MaxBet:       decimal.NewFromInt(1000),
		}

		isActive := true
		existingCountry := &models.Country{
			ID:       uuid.New(),
			Code:     "NGA",
			IsActive: &isActive,
		}

		mockRepo.On("GetByCode", ctx, "NGA").Return(existingCountry, nil)

		result, err := srvc.CreateCountry(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "country with this code already exists")
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on Check", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		req := &CreateCountryRequest{
			Name:         "Nigeria",
			Code:         "NGA",
			CurrencyCode: "NGN",
			MinBet:       decimal.NewFromInt(10),
			MaxBet:       decimal.NewFromInt(1000),
		}

		mockRepo.On("GetByCode", ctx, "NGA").Return(nil, assert.AnError)

		result, err := srvc.CreateCountry(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on Create", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()

		req := &CreateCountryRequest{
			Name:           "Nigeria",
			Code:           "NGA",
			CurrencyCode:   "NGN",
			CurrencySymbol: "₦",
			MinBet:         decimal.NewFromInt(10),
			MaxBet:         decimal.NewFromInt(1000),
		}

		mockRepo.On("GetByCode", ctx, "NGA").Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Country")).Return(assert.AnError)

		result, err := srvc.CreateCountry(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_UpdateCountry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		isActive := true
		existingCountry := &models.Country{
			ID:             id,
			Name:           "Nigeria",
			Code:           "NGA",
			CurrencySymbol: "₦",
			CurrencyCode:   "NGN",
			IsActive:       &isActive,
			Config: &models.CountryConfig{
				MinBet:      decimal.NewFromInt(10),
				MaxBet:      decimal.NewFromInt(1000),
				KYCRequired: false,
			},
		}

		newName := "Updated Nigeria"
		newKYC := true
		currencySymbol := "₦"
		timezone := "Africa/Lagos"
		req := UpdateCountryRequest{
			Name:           &newName,
			KYCRequired:    &newKYC,
			CurrencySymbol: &currencySymbol,
			IsActive:       &isActive,
			Timezone:       &timezone,
			ContractUnit:   &existingCountry.Config.ContractUnit,
			MinBet:         &existingCountry.Config.MinBet,
			MaxBet:         &existingCountry.Config.MaxBet,
		}

		mockRepo.On("GetByID", ctx, id).Return(existingCountry, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Country")).Return(nil)

		result, err := srvc.UpdateCountry(ctx, id, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Country Not Found", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		req := UpdateCountryRequest{}

		mockRepo.On("GetByID", ctx, id).Return(nil, gorm.ErrRecordNotFound)

		result, err := srvc.UpdateCountry(ctx, id, req)

		assert.Error(t, err)
		assert.Equal(t, models.ErrRecordNotFound, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Min/Max Bet Update", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		isActive := true
		existingCountry := &models.Country{
			ID:       id,
			IsActive: &isActive,
			Config: &models.CountryConfig{
				MinBet: decimal.NewFromInt(10),
				MaxBet: decimal.NewFromInt(1000),
			},
		}

		newMinBet := decimal.NewFromInt(2000)
		req := UpdateCountryRequest{
			MinBet: &newMinBet, // This will make min > max
		}

		mockRepo.On("GetByID", ctx, id).Return(existingCountry, nil)

		result, err := srvc.UpdateCountry(ctx, id, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "min_bet must be less than max_bet")
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_DeleteCountry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		isActive := true
		country := &models.Country{
			ID:       id,
			IsActive: &isActive,
		}

		mockRepo.On("GetByID", ctx, id).Return(country, nil)
		mockRepo.On("Delete", ctx, id).Return(nil)

		err := srvc.DeleteCountry(ctx, id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Country Not Found", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		mockRepo.On("GetByID", ctx, id).Return(nil, gorm.ErrRecordNotFound)

		err := srvc.DeleteCountry(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, models.ErrRecordNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on Check", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		mockRepo.On("GetByID", ctx, id).Return(nil, assert.AnError)

		err := srvc.DeleteCountry(ctx, id)

		assert.Error(t, err)
		assert.NotEqual(t, models.ErrRecordNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on Delete", func(t *testing.T) {
		mockRepo := new(mocks.MockCountryRepository)
		srvc := NewService(mockRepo)
		ctx := context.Background()
		id := uuid.New()

		isActive := true
		country := &models.Country{
			ID:       id,
			IsActive: &isActive,
		}

		mockRepo.On("GetByID", ctx, id).Return(country, nil)
		mockRepo.On("Delete", ctx, id).Return(assert.AnError)

		err := srvc.DeleteCountry(ctx, id)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
