package prediction

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
)

// repository implements the Repository interface using GORM
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new betting repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) WithTx(tx *gorm.DB) Repository {
	return &repository{db: tx}
}

// GetBetByID returns a bet by ID with related data
func (r *repository) GetBetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error) {
	var bet models.Bet
	err := r.db.WithContext(ctx).
		Preload("Market").
		Preload("MarketOutcome").
		Preload("User").
		Preload("Transaction").
		Where("id = ?", id).
		First(&bet).Error
	if err != nil {
		return nil, err
	}
	return &bet, nil
}

// GetBetsByUser returns paginated bets for a user with filters
func (r *repository) GetBetsByUser(ctx context.Context, userID uuid.UUID, filters *BetFilters) ([]models.Bet, int64, error) {
	var bets []models.Bet
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Bet{}).Where("user_id = ?", userID)

	// Apply filters
	query = r.applyBetFilters(query, filters)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and sorting
	query = r.applyBetSorting(query, filters)
	query = r.applyBetPagination(query, filters)

	// Preload related data
	query = query.Preload("Market").Preload("MarketOutcome").Preload("Transaction")

	err := query.Find(&bets).Error
	return bets, total, err
}

// GetBetsByMarket returns all bets for a market
func (r *repository) GetBetsByMarket(ctx context.Context, marketID uuid.UUID) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("MarketOutcome").
		Where("market_id = ? AND status = ?", marketID, models.BetStatusActive).
		Order("created_at DESC").
		Find(&bets).Error
	return bets, err
}

// GetActiveBetsByUser returns all active bets for a user
func (r *repository) GetActiveBetsByUser(ctx context.Context, userID uuid.UUID) ([]models.Bet, error) {
	var bets []models.Bet
	err := r.db.WithContext(ctx).
		Preload("Market").
		Preload("MarketOutcome").
		Where("user_id = ? AND status = ?", userID, models.BetStatusActive).
		Order("created_at DESC").
		Find(&bets).Error
	return bets, err
}

// CreateBet creates a new bet in a transaction
func (r *repository) CreateBet(ctx context.Context, bet *models.Bet) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(bet).Error
	})
}

// UpdateBet updates an existing bet
func (r *repository) UpdateBet(ctx context.Context, bet *models.Bet) error {
	return r.db.WithContext(ctx).Save(bet).Error
}

// GetUserPositionInMarket calculates user's total position in a market
func (r *repository) GetUserPositionInMarket(ctx context.Context, userID, marketID uuid.UUID) (decimal.Decimal, error) {
	var totalAmount decimal.Decimal
	err := r.db.WithContext(ctx).
		Model(&models.Bet{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND market_id = ? AND status = ?", userID, marketID, models.BetStatusActive).
		Scan(&totalAmount).Error
	return totalAmount, err
}

// GetUserDailyBetAmount calculates user's total bet amount for a day
func (r *repository) GetUserDailyBetAmount(ctx context.Context, userID uuid.UUID, date time.Time) (decimal.Decimal, error) {
	var totalAmount decimal.Decimal
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.WithContext(ctx).
		Model(&models.Bet{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startOfDay, endOfDay).
		Scan(&totalAmount).Error
	return totalAmount, err
}

// GetUserBetCount returns the number of bets placed by user since a time
func (r *repository) GetUserBetCount(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Bet{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&count).Error
	return int(count), err
}

// GetMarketWithOutcomes returns a market with its outcomes
func (r *repository) GetMarketWithOutcomes(ctx context.Context, marketID uuid.UUID) (*models.Market, error) {
	var market models.Market
	err := r.db.WithContext(ctx).
		Preload("Outcomes").
		Where("id = ?", marketID).
		First(&market).Error
	if err != nil {
		return nil, err
	}
	return &market, nil
}

// GetMarketOutcome returns a market outcome by ID
func (r *repository) GetMarketOutcome(ctx context.Context, outcomeID uuid.UUID) (*models.MarketOutcome, error) {
	var outcome models.MarketOutcome
	err := r.db.WithContext(ctx).
		Preload("Market").
		Where("id = ?", outcomeID).
		First(&outcome).Error
	if err != nil {
		return nil, err
	}
	return &outcome, nil
}

// UpdateMarketOutcome updates a market outcome
func (r *repository) UpdateMarketOutcome(ctx context.Context, outcome *models.MarketOutcome) error {
	return r.db.WithContext(ctx).Save(outcome).Error
}

// UpdateMarket updates a market
func (r *repository) UpdateMarket(ctx context.Context, market *models.Market) error {
	return r.db.WithContext(ctx).Save(market).Error
}

// GetUserWallet returns user's wallet for a currency
func (r *repository) GetUserWallet(ctx context.Context, userID uuid.UUID, currencyCode string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND currency_code = ?", userID, currencyCode).
		First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

// UpdateWallet updates a user's wallet
func (r *repository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}

// CreateTransaction creates a new transaction
func (r *repository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// Helper methods for filtering, sorting, and pagination

func (r *repository) applyBetFilters(query *gorm.DB, filters *BetFilters) *gorm.DB {
	if filters.MarketID != nil {
		query = query.Where("market_id = ?", *filters.MarketID)
	}

	if filters.OutcomeID != nil {
		query = query.Where("market_outcome_id = ?", *filters.OutcomeID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.DateFrom != nil {
		query = query.Where("created_at >= ?", *filters.DateFrom)
	}

	if filters.DateTo != nil {
		query = query.Where("created_at <= ?", *filters.DateTo)
	}

	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}

	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}

	return query
}

func (r *repository) applyBetSorting(query *gorm.DB, filters *BetFilters) *gorm.DB {
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at" // Default sort
	}

	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "desc" // Default order
	}

	// Validate sort fields to prevent SQL injection
	validSortFields := map[string]bool{
		"created_at":         true,
		"amount":             true,
		"price_per_contract": true,
		"total_cost":         true,
		"contracts_bought":   true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
	return query.Order(orderClause)
}

func (r *repository) applyBetPagination(query *gorm.DB, filters *BetFilters) *gorm.DB {
	page := filters.Page
	if page < 1 {
		page = 1
	}

	perPage := filters.PerPage
	if perPage < 1 || perPage > 100 { // Limit max per page
		perPage = 20 // Default per page
	}

	offset := (page - 1) * perPage
	return query.Offset(offset).Limit(perPage)
}

func (r *repository) UpdateTransaction(ctx context.Context, transaction *models.Transaction) error {
	return r.db.WithContext(ctx).Save(transaction).Error
}
