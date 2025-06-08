package models

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCategory(t *testing.T) {
	t.Run("TableName", func(t *testing.T) {
		c := Category{}
		assert.Equal(t, "categories", c.TableName())
	})

	t.Run("BeforeCreate", func(t *testing.T) {
		c := Category{}
		assert.Equal(t, uuid.Nil, c.ID)

		err := c.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, c.ID)

		existingID := uuid.New()
		c2 := Category{ID: existingID}
		err = c2.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, c2.ID)
	})

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name        string
			category    Category
			expectedErr error
		}{
			{
				name: "Valid category",
				category: Category{
					CountryID: uuid.New(),
					Name:      "Sports",
					Slug:      "sports",
				},
				expectedErr: nil,
			},
			{
				name: "Invalid country ID",
				category: Category{
					CountryID: uuid.Nil,
					Name:      "Sports",
					Slug:      "sports",
				},
				expectedErr: ErrInvalidCountryID,
			},
			{
				name: "Empty name",
				category: Category{
					CountryID: uuid.New(),
					Name:      "",
					Slug:      "sports",
				},
				expectedErr: ErrInvalidCategoryName,
			},
			{
				name: "Empty slug",
				category: Category{
					CountryID: uuid.New(),
					Name:      "Sports",
					Slug:      "",
				},
				expectedErr: ErrInvalidCategorySlug,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.category.Validate()
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("IsValidSlug", func(t *testing.T) {
		tests := []struct {
			name     string
			slug     string
			expected bool
		}{
			{"Valid lowercase", "sports", true},
			{"Valid with numbers", "sports123", true},
			{"Valid with hyphens", "sports-betting", true},
			{"Valid complex", "sports-betting-123", true},
			{"Empty slug", "", false},
			{"Uppercase letters", "Sports", false},
			{"With spaces", "sports betting", false},
			{"With underscores", "sports_betting", false},
			{"With special chars", "sports@betting", false},
			{"With dots", "sports.betting", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := Category{Slug: tt.slug}
				assert.Equal(t, tt.expected, c.IsValidSlug())
			})
		}
	})

	t.Run("Full integration", func(t *testing.T) {
		countryID := uuid.New()
		c := Category{
			ID:          uuid.New(),
			CountryID:   countryID,
			Name:        "Sports Betting",
			Slug:        "sports-betting",
			Description: "All sports related markets",
			IsActive:    true,
			SortOrder:   1,
		}

		assert.NotEmpty(t, c.ID)
		assert.True(t, c.IsValidSlug())
		assert.NoError(t, c.Validate())
		assert.Equal(t, "categories", c.TableName())
	})

	t.Run("GetActiveMarkets", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		categoryID := uuid.New()
		category := Category{ID: categoryID}

		rows := sqlmock.NewRows([]string{"id", "category_id", "status"}).
			AddRow(uuid.New(), categoryID, "open").
			AddRow(uuid.New(), categoryID, "closed")

		mock.ExpectQuery(`SELECT \* FROM "markets" WHERE category_id = \$1 AND status IN \(\$2,\$3\)`).
			WithArgs(categoryID, "open", "closed").
			WillReturnRows(rows)

		markets, err := category.GetActiveMarkets(gormDB)
		assert.NoError(t, err)
		assert.Len(t, markets, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetMarketCount", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		categoryID := uuid.New()
		category := Category{ID: categoryID}

		rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "markets" WHERE category_id = \$1`).
			WithArgs(categoryID).
			WillReturnRows(rows)

		count, err := category.GetMarketCount(gormDB)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetMarketCount empty category", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		assert.NoError(t, err)

		categoryID := uuid.New()
		category := Category{ID: categoryID}

		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "markets" WHERE category_id = \$1`).
			WithArgs(categoryID).
			WillReturnRows(rows)

		count, err := category.GetMarketCount(gormDB)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
