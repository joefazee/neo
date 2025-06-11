package wallet

import (
	"context"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"gorm.io/gorm"
)

type Repository interface {
	CreateWallet(ctx context.Context, wallet *models.Wallet) error
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currencyCode string) (*models.Wallet, error)
	GetUserWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error)
	UpdateWallet(ctx context.Context, wallet *models.Wallet) error

	CreateTransaction(ctx context.Context, transaction *models.Transaction) error
	GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]models.Transaction, error)

	WithTx(tx *gorm.DB) Repository
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) Repository {
	return &repository{db: tx}
}

func (r *repository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *repository) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *repository) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currencyCode string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND currency_code = ?", userID, currencyCode).
		First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *repository) GetUserWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error) {
	var wallets []models.Wallet
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("currency_code ASC").
		Find(&wallets).Error
	return wallets, err
}

func (r *repository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(wallet).Error
}

func (r *repository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	if err := transaction.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(transaction).Error
}

func (r *repository) GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error
	return transactions, err
}
