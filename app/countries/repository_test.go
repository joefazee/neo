package countries

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
	"github.com/joefazee/neo/tests/suites"
	"github.com/stretchr/testify/suite"
)

type CountriesRepositoryTestSuite struct {
	suites.RepositoryTestSuite
	repo Repository
}

func (suite *CountriesRepositoryTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping database integration test")
	}

	suite.AutoMigrate = true

	suite.RepositoryTestSuite.SetupSuite()

	suite.repo = NewRepository(suite.DB)
}

func TestCountriesRepository(t *testing.T) {
	suite.Run(t, new(CountriesRepositoryTestSuite))
}

func (suite *CountriesRepositoryTestSuite) TestCreate() {
	ctx := context.Background()

	country := &models.Country{
		Name:           "Nigeria",
		Code:           "NGA",
		CurrencyCode:   "NGN",
		CurrencySymbol: "₦",
	}

	err := suite.repo.Create(ctx, country)
	assert.NoError(suite.T(), err, "Failed to create country")
}

func (suite *CountriesRepositoryTestSuite) TestGetAll() {
	ctx := context.Background()
	suite.seedCountries()

	countries, err := suite.repo.GetAll(ctx)
	suite.AssertNoDBError(err)
	suite.Assert().Len(countries, 3)
}

func (suite *CountriesRepositoryTestSuite) TestGetByID() {
	ctx := context.Background()
	createdCountry := suite.createTestCountry("USA", "USD")

	country, err := suite.repo.GetByID(ctx, createdCountry.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal("USA", country.Code)
	suite.Assert().Equal(createdCountry.ID, country.ID)
}

func (suite *CountriesRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	country, err := suite.repo.GetByID(ctx, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().Nil(country)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *CountriesRepositoryTestSuite) TestGetByCode() {
	ctx := context.Background()
	suite.createTestCountry("GBR", "GBP")

	country, err := suite.repo.GetByCode(ctx, "GBR")
	suite.AssertNoDBError(err)
	suite.Assert().Equal("GBR", country.Code)
	suite.Assert().Equal("GBP", country.CurrencyCode)
}

func (suite *CountriesRepositoryTestSuite) TestGetByCode_NotFound() {
	ctx := context.Background()

	country, err := suite.repo.GetByCode(ctx, "XYZ")
	suite.AssertDBError(err)
	suite.Assert().Nil(country)
}

func (suite *CountriesRepositoryTestSuite) TestGetActive() {
	ctx := context.Background()

	usa := suite.createTestCountryWithStatus("USA", true)
	suite.Assert().NotNil(usa)

	gbr := suite.createTestCountryWithStatus("GBR", true)
	suite.Assert().NotNil(gbr)

	due := suite.createTestCountryWithStatus("DEU", false)
	suite.Assert().NotNil(due)

	activeCountries, err := suite.repo.GetActive(ctx)
	suite.AssertNoDBError(err)
	suite.Assert().Len(activeCountries, 2)

	for i := range activeCountries {
		country := activeCountries[i]
		suite.Assert().True(*country.IsActive)
	}
}

func (suite *CountriesRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	country := suite.createTestCountry("FRA", "EUR")

	country.Name = "France Updated"
	*country.IsActive = false

	err := suite.repo.Update(ctx, country)
	suite.AssertNoDBError(err)

	updated, err := suite.repo.GetByID(ctx, country.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal("France Updated", updated.Name)
	suite.Assert().False(*updated.IsActive)
}

func (suite *CountriesRepositoryTestSuite) TestDelete() {
	ctx := context.Background()
	country := suite.createTestCountry("DEL", "DEL")

	err := suite.repo.Delete(ctx, country.ID)
	suite.AssertNoDBError(err)

	_, err = suite.repo.GetByID(ctx, country.ID)
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)

	count := suite.CountRecords("countries")
	suite.Assert().Equal(int64(0), count)
}

func (suite *CountriesRepositoryTestSuite) createTestCountry(code, currency string) *models.Country {
	isActive := true
	country := &models.Country{
		Name:           "Test " + code,
		Code:           code,
		CurrencyCode:   currency,
		CurrencySymbol: "$",
		IsActive:       &isActive,
	}
	err := suite.repo.Create(context.Background(), country)
	suite.AssertNoDBError(err)
	return country
}
func (suite *CountriesRepositoryTestSuite) createTestCountryWithStatus(code string, isActive bool) *models.Country {
	country := &models.Country{
		Name:           "Test " + code,
		Code:           code,
		CurrencyCode:   "USD",
		CurrencySymbol: "$",
		IsActive:       &isActive,
	}
	err := suite.repo.Create(context.Background(), country)
	suite.AssertNoDBError(err)
	return country
}

func (suite *CountriesRepositoryTestSuite) seedCountries() {
	isActive := true
	countries := []*models.Country{
		{Name: "United States", Code: "USA", CurrencyCode: "USD", CurrencySymbol: "$", IsActive: &isActive},
		{Name: "United Kingdom", Code: "GBR", CurrencyCode: "GBP", CurrencySymbol: "£", IsActive: &isActive},
		{Name: "Germany", Code: "DEU", CurrencyCode: "EUR", CurrencySymbol: "€", IsActive: &isActive},
	}

	for _, country := range countries {
		err := suite.repo.Create(context.Background(), country)
		suite.AssertNoDBError(err)
	}
}
