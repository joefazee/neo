package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service interface {
	CreateWallet(ctx context.Context, req *CreateWalletRequest) (*Response, error)
	GetWallet(ctx context.Context, id uuid.UUID) (*Response, error)
	GetUserWallets(ctx context.Context, userID uuid.UUID) ([]Response, error)

	LockWallet(ctx context.Context, id uuid.UUID, req *LockWalletRequest) (*OperationResponse, error)
	CreditWallet(ctx context.Context, id uuid.UUID, req *CreditWalletRequest) (*OperationResponse, error)
	DebitWallet(ctx context.Context, id uuid.UUID, req *DebitWalletRequest) (*OperationResponse, error)
	LockFunds(ctx context.Context, id uuid.UUID, req *LockFundsRequest) (*OperationResponse, error)
	UnlockFunds(ctx context.Context, id uuid.UUID, req *UnlockFundsRequest) (*OperationResponse, error)

	GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]TransactionResponse, error)
}

type service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return &service{
		repo: repo,
		db:   db,
	}
}

func (s *service) CreateWallet(ctx context.Context, req *CreateWalletRequest) (*Response, error) {
	// Check if wallet already exists
	existing, err := s.repo.GetWalletByUserAndCurrency(ctx, req.UserID, req.CurrencyCode)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing wallet: %w", err)
	}
	if existing != nil {
		return nil, errors.New("wallet already exists for this user and currency")
	}

	wallet := &models.Wallet{
		UserID:        req.UserID,
		CurrencyCode:  req.CurrencyCode,
		Balance:       decimal.Zero,
		LockedBalance: decimal.Zero,
	}

	if err := s.repo.CreateWallet(ctx, wallet); err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return ToWalletResponse(wallet), nil
}

func (s *service) GetWallet(ctx context.Context, id uuid.UUID) (*Response, error) {
	wallet, err := s.repo.GetWalletByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return ToWalletResponse(wallet), nil
}

func (s *service) GetUserWallets(ctx context.Context, userID uuid.UUID) ([]Response, error) {
	wallets, err := s.repo.GetUserWallets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user wallets: %w", err)
	}

	responses := make([]Response, len(wallets))
	for i := range wallets {
		responses[i] = *ToWalletResponse(&wallets[i])
	}

	return responses, nil
}

func (s *service) LockWallet(ctx context.Context, id uuid.UUID, req *LockWalletRequest) (*OperationResponse, error) {
	return s.executeWalletTransaction(func(txRepo Repository) (*OperationResponse, error) {
		wallet, err := txRepo.GetWalletByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, models.ErrRecordNotFound
			}
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if req.IsLocked {
			wallet.Lock(req.Reason)
		} else {
			wallet.Unlock()
		}

		if err := txRepo.UpdateWallet(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		return &OperationResponse{
			Wallet: ToWalletResponse(wallet),
		}, nil
	})
}

func (s *service) CreditWallet(ctx context.Context, id uuid.UUID, req *CreditWalletRequest) (*OperationResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("credit amount must be positive")
	}

	return s.executeWalletTransaction(func(txRepo Repository) (*OperationResponse, error) {
		wallet, err := txRepo.GetWalletByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, models.ErrRecordNotFound
			}
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if !wallet.IsOperationAllowed() {
			return nil, errors.New("wallet is locked")
		}

		balanceBefore := wallet.Balance

		if err := wallet.Credit(req.Amount); err != nil {
			return nil, fmt.Errorf("failed to credit wallet: %w", err)
		}

		// Create transaction record
		transaction := &models.Transaction{
			UserID:          wallet.UserID,
			WalletID:        wallet.ID,
			TransactionType: models.TransactionTypeDeposit,
			Amount:          req.Amount,
			BalanceBefore:   balanceBefore,
			BalanceAfter:    wallet.Balance,
			ReferenceType:   "credit",
			ReferenceID:     req.ReferenceID,
			Description:     req.Description,
		}

		// Update wallet and create transaction atomically
		if err := txRepo.UpdateWallet(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		if err := txRepo.CreateTransaction(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		return &OperationResponse{
			Wallet:      ToWalletResponse(wallet),
			Transaction: ToTransactionResponse(transaction),
		}, nil
	})
}

func (s *service) DebitWallet(ctx context.Context, id uuid.UUID, req *DebitWalletRequest) (*OperationResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("debit amount must be positive")
	}

	return s.executeWalletTransaction(func(txRepo Repository) (*OperationResponse, error) {
		wallet, err := txRepo.GetWalletByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, models.ErrRecordNotFound
			}
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if !wallet.IsOperationAllowed() {
			return nil, errors.New("wallet is locked")
		}

		balanceBefore := wallet.Balance

		// Check sufficient balance
		if !wallet.CanDebit(req.Amount) {
			return nil, models.ErrInsufficientBalance
		}

		// Debit the wallet
		if err := wallet.Debit(req.Amount); err != nil {
			return nil, fmt.Errorf("failed to debit wallet: %w", err)
		}

		// Create transaction record
		transaction := &models.Transaction{
			UserID:          wallet.UserID,
			WalletID:        wallet.ID,
			TransactionType: models.TransactionTypeWithdrawal,
			Amount:          req.Amount.Neg(), // Negative for debit
			BalanceBefore:   balanceBefore,
			BalanceAfter:    wallet.Balance,
			ReferenceType:   "debit",
			ReferenceID:     req.ReferenceID,
			Description:     req.Description,
		}

		// Update wallet and create transaction atomically
		if err := txRepo.UpdateWallet(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		if err := txRepo.CreateTransaction(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		return &OperationResponse{
			Wallet:      ToWalletResponse(wallet),
			Transaction: ToTransactionResponse(transaction),
		}, nil
	})
}

// LockFunds locks a specified amount of funds in the wallet
// //nolint: dupl
func (s *service) LockFunds(ctx context.Context, id uuid.UUID, req *LockFundsRequest) (*OperationResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("lock amount must be positive")
	}

	return s.executeWalletTransaction(func(txRepo Repository) (*OperationResponse, error) {
		wallet, err := txRepo.GetWalletByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, models.ErrRecordNotFound
			}
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if !wallet.IsOperationAllowed() {
			return nil, errors.New("wallet is locked")
		}

		// Lock funds
		if err := wallet.LockFunds(req.Amount); err != nil {
			return nil, fmt.Errorf("failed to lock funds: %w", err)
		}

		// Create transaction record for audit trail
		transaction := &models.Transaction{
			UserID:          wallet.UserID,
			WalletID:        wallet.ID,
			TransactionType: "fund_lock",  // Custom type for fund locking
			Amount:          decimal.Zero, // No balance change, just locking
			BalanceBefore:   wallet.Balance,
			BalanceAfter:    wallet.Balance,
			ReferenceType:   "lock",
			ReferenceID:     req.ReferenceID,
			Description:     fmt.Sprintf("Locked %s - %s", req.Amount.String(), req.Description),
		}

		// Update wallet and create transaction atomically
		if err := txRepo.UpdateWallet(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		if err := txRepo.CreateTransaction(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		return &OperationResponse{
			Wallet:      ToWalletResponse(wallet),
			Transaction: ToTransactionResponse(transaction),
		}, nil
	})
}

// UnlockFunds unlocks a specified amount of funds in the wallet
// //nolint:dupl
func (s *service) UnlockFunds(ctx context.Context, id uuid.UUID, req *UnlockFundsRequest) (*OperationResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("unlock amount must be positive")
	}

	return s.executeWalletTransaction(func(txRepo Repository) (*OperationResponse, error) {
		wallet, err := txRepo.GetWalletByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, models.ErrRecordNotFound
			}
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if !wallet.IsOperationAllowed() {
			return nil, errors.New("wallet is locked")
		}

		// Unlock funds
		if err := wallet.UnlockFunds(req.Amount); err != nil {
			return nil, fmt.Errorf("failed to unlock funds: %w", err)
		}

		// Create transaction record for audit trail
		transaction := &models.Transaction{
			UserID:          wallet.UserID,
			WalletID:        wallet.ID,
			TransactionType: "fund_unlock", // Custom type for fund unlocking
			Amount:          decimal.Zero,  // No balance change, just unlocking
			BalanceBefore:   wallet.Balance,
			BalanceAfter:    wallet.Balance,
			ReferenceType:   "unlock",
			ReferenceID:     req.ReferenceID,
			Description:     fmt.Sprintf("Unlocked %s - %s", req.Amount.String(), req.Description),
		}

		// Update wallet and create transaction atomically
		if err := txRepo.UpdateWallet(ctx, wallet); err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		if err := txRepo.CreateTransaction(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		return &OperationResponse{
			Wallet:      ToWalletResponse(wallet),
			Transaction: ToTransactionResponse(transaction),
		}, nil
	})
}

func (s *service) GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]TransactionResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	transactions, err := s.repo.GetWalletTransactions(ctx, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet transactions: %w", err)
	}

	responses := make([]TransactionResponse, len(transactions))
	for i := range transactions {
		transaction := transactions[i]
		responses[i] = *ToTransactionResponse(&transaction)
	}

	return responses, nil
}

// executeWalletTransaction executes a wallet operation within a database transaction
func (s *service) executeWalletTransaction(operation func(Repository) (*OperationResponse, error)) (*OperationResponse, error) {
	var result *OperationResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)

		var err error
		result, err = operation(txRepo)
		return err
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
