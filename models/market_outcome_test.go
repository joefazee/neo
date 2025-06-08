package models

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMarketOutcome(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		mo := MarketOutcome{}
		assert.Equal(t, "market_outcomes", mo.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		mo := MarketOutcome{}
		assert.Equal(t, uuid.Nil, mo.ID)

		err := mo.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, mo.ID)

		existingID := uuid.New()
		mo2 := MarketOutcome{ID: existingID}
		err = mo2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, mo2.ID)
	})

	t.Run("GetCurrentPrice", func(t *testing.T) {
		mo := MarketOutcome{PoolAmount: decimal.NewFromFloat(300)}

		totalPool := decimal.NewFromFloat(1000)
		price := mo.GetCurrentPrice(totalPool)
		expected := decimal.NewFromFloat(30) // 300/1000 * 100
		assert.True(t, expected.Equal(price))

		price = mo.GetCurrentPrice(decimal.Zero)
		expected = decimal.NewFromFloat(50)
		assert.True(t, expected.Equal(price))

		mo.PoolAmount = decimal.NewFromFloat(5)
		price = mo.GetCurrentPrice(decimal.NewFromFloat(1000))
		expected = decimal.NewFromFloat(1)
		assert.True(t, expected.Equal(price))

		mo.PoolAmount = decimal.NewFromFloat(995)
		price = mo.GetCurrentPrice(decimal.NewFromFloat(1000))
		expected = decimal.NewFromFloat(99)
		assert.True(t, expected.Equal(price))
	})

	t.Run("AddToPool", func(t *testing.T) {
		mo := MarketOutcome{PoolAmount: decimal.NewFromFloat(100)}

		err := mo.AddToPool(decimal.NewFromFloat(50))
		assert.NoError(t, err)
		expected := decimal.NewFromFloat(150)
		assert.True(t, expected.Equal(mo.PoolAmount))

		err = mo.AddToPool(decimal.Zero)
		assert.Equal(t, ErrInvalidBetAmount, err)

		err = mo.AddToPool(decimal.NewFromFloat(-10))
		assert.Equal(t, ErrInvalidBetAmount, err)
	})

	t.Run("Winner/Loser operations", func(t *testing.T) {
		mo := MarketOutcome{}

		assert.True(t, mo.IsUnresolved())
		assert.False(t, mo.IsWinner())
		assert.False(t, mo.IsLoser())

		mo.SetAsWinner()
		assert.True(t, mo.IsWinner())
		assert.False(t, mo.IsLoser())
		assert.False(t, mo.IsUnresolved())

		mo.SetAsLoser()
		assert.False(t, mo.IsWinner())
		assert.True(t, mo.IsLoser())
		assert.False(t, mo.IsUnresolved())
	})

	t.Run("Validate", func(t *testing.T) {
		validOutcome := MarketOutcome{
			MarketID:     uuid.New(),
			OutcomeKey:   "yes",
			OutcomeLabel: "Yes",
			PoolAmount:   decimal.NewFromFloat(100),
		}
		assert.NoError(t, validOutcome.Validate())

		tests := []struct {
			name   string
			modify func(*MarketOutcome)
			err    error
		}{
			{"Invalid MarketID", func(mo *MarketOutcome) { mo.MarketID = uuid.Nil }, ErrInvalidMarketID},
			{"Empty OutcomeKey", func(mo *MarketOutcome) { mo.OutcomeKey = "" }, ErrInvalidOutcomeKey},
			{"Empty OutcomeLabel", func(mo *MarketOutcome) { mo.OutcomeLabel = "" }, ErrInvalidOutcomeLabel},
			{"Negative PoolAmount", func(mo *MarketOutcome) { mo.PoolAmount = decimal.NewFromFloat(-10) }, ErrInvalidBetAmount},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				outcome := validOutcome
				tt.modify(&outcome)
				assert.Equal(t, tt.err, outcome.Validate())
			})
		}
	})

	t.Run("GetBetCount", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		outcomeID := uuid.New()
		mo := MarketOutcome{ID: outcomeID}

		rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "bets" WHERE market_outcome_id = \$1 AND status = \$2`).
			WithArgs(outcomeID, "active").
			WillReturnRows(rows)

		count, err := mo.GetBetCount(gormDB)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUniqueBettors", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		outcomeID := uuid.New()
		mo := MarketOutcome{ID: outcomeID}

		rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT\("user_id"\)\) FROM "bets" WHERE market_outcome_id = \$1 AND status = \$2`).
			WithArgs(outcomeID, "active").
			WillReturnRows(rows)

		count, err := mo.GetUniqueBettors(gormDB)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Full integration", func(t *testing.T) {
		mo := MarketOutcome{
			ID:           uuid.New(),
			MarketID:     uuid.New(),
			OutcomeKey:   "yes",
			OutcomeLabel: "Yes",
			SortOrder:    1,
			PoolAmount:   decimal.NewFromFloat(250),
		}

		assert.NotEmpty(t, mo.ID)
		assert.NoError(t, mo.Validate())
		assert.True(t, mo.IsUnresolved())

		totalPool := decimal.NewFromFloat(1000)
		price := mo.GetCurrentPrice(totalPool)
		expected := decimal.NewFromFloat(25)
		assert.True(t, expected.Equal(price))

		mo.SetAsWinner()
		assert.True(t, mo.IsWinner())
		assert.False(t, mo.IsUnresolved())
	})
}
