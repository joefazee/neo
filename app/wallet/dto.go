package wallet

import (
	"time"

	"github.com/joefazee/neo/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateWalletRequest represents the request to create a wallet
type CreateWalletRequest struct {
	UserID       uuid.UUID `json:"user_id" binding:"required"`
	CurrencyCode string    `json:"currency_code" binding:"required,len=3"`
}

// CreditWalletRequest represents the request to credit a wallet
type CreditWalletRequest struct {
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Description string          `json:"description,omitempty"`
	ReferenceID *uuid.UUID      `json:"reference_id,omitempty"`
}

// DebitWalletRequest represents the request to debit a wallet
type DebitWalletRequest struct {
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Description string          `json:"description,omitempty"`
	ReferenceID *uuid.UUID      `json:"reference_id,omitempty"`
}

// LockFundsRequest represents the request to lock funds
type LockFundsRequest struct {
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Description string          `json:"description,omitempty"`
	ReferenceID *uuid.UUID      `json:"reference_id,omitempty"`
}

// UnlockFundsRequest represents the request to unlock funds
type UnlockFundsRequest struct {
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Description string          `json:"description,omitempty"`
	ReferenceID *uuid.UUID      `json:"reference_id,omitempty"`
}

// LockWalletRequest represents the request to lock/unlock a wallet
type LockWalletRequest struct {
	IsLocked bool   `json:"is_locked"`
	Reason   string `json:"reason,omitempty"`
}

// Response represents a wallet in API responses
type Response struct {
	ID               uuid.UUID       `json:"id"`
	UserID           uuid.UUID       `json:"user_id"`
	CurrencyCode     string          `json:"currency_code"`
	Balance          decimal.Decimal `json:"balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	IsLocked         bool            `json:"is_locked"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// TransactionResponse represents a transaction in API responses
type TransactionResponse struct {
	ID              uuid.UUID                  `json:"id"`
	UserID          uuid.UUID                  `json:"user_id"`
	WalletID        uuid.UUID                  `json:"wallet_id"`
	TransactionType models.TransactionType     `json:"transaction_type"`
	Amount          decimal.Decimal            `json:"amount"`
	BalanceBefore   decimal.Decimal            `json:"balance_before"`
	BalanceAfter    decimal.Decimal            `json:"balance_after"`
	ReferenceType   string                     `json:"reference_type"`
	ReferenceID     *uuid.UUID                 `json:"reference_id"`
	Description     string                     `json:"description"`
	Metadata        models.TransactionMetadata `json:"metadata"`
	CreatedAt       time.Time                  `json:"created_at"`
}

// OperationResponse represents the response for wallet operations
type OperationResponse struct {
	Wallet      *Response            `json:"wallet"`
	Transaction *TransactionResponse `json:"transaction"`
}

// ToWalletResponse converts a models.Wallet to WalletResponse
func ToWalletResponse(wallet *models.Wallet) *Response {
	return &Response{
		ID:               wallet.ID,
		UserID:           wallet.UserID,
		CurrencyCode:     wallet.CurrencyCode,
		Balance:          wallet.Balance,
		LockedBalance:    wallet.LockedBalance,
		AvailableBalance: wallet.GetAvailableBalance(),
		IsLocked:         wallet.IsLocked,
		CreatedAt:        wallet.CreatedAt,
		UpdatedAt:        wallet.UpdatedAt,
	}
}

// ToTransactionResponse converts a models.Transaction to TransactionResponse
func ToTransactionResponse(transaction *models.Transaction) *TransactionResponse {
	return &TransactionResponse{
		ID:              transaction.ID,
		UserID:          transaction.UserID,
		WalletID:        transaction.WalletID,
		TransactionType: transaction.TransactionType,
		Amount:          transaction.Amount,
		BalanceBefore:   transaction.BalanceBefore,
		BalanceAfter:    transaction.BalanceAfter,
		ReferenceType:   transaction.ReferenceType,
		ReferenceID:     transaction.ReferenceID,
		Description:     transaction.Description,
		Metadata:        transaction.Metadata,
		CreatedAt:       transaction.CreatedAt,
	}
}
