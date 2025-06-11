package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Wallet represents a user's wallet for a specific currency
type Wallet struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index:idx_wallets_user_currency" json:"user_id"`
	CurrencyCode  string          `gorm:"type:varchar(3);not null;index:idx_wallets_user_currency" json:"currency_code"`
	Balance       decimal.Decimal `gorm:"type:decimal(20,2);default:0.00;check:balance >= 0" json:"balance"`
	LockedBalance decimal.Decimal `gorm:"type:decimal(20,2);default:0.00;check:locked_balance >= 0" json:"locked_balance"`
	IsLocked      bool            `gorm:"default:false" json:"is_locked"`
	LockReason    string          `gorm:"type:text" json:"lock_reason,omitempty"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	User         *User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Transactions []Transaction `gorm:"foreignKey:WalletID" json:"-"`
}

// TableName specifies the table name for Wallet model
func (*Wallet) TableName() string {
	return "wallets"
}

// BeforeCreate sets up the model before creation
func (w *Wallet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// GetAvailableBalance returns the available balance (total - locked)
func (w *Wallet) GetAvailableBalance() decimal.Decimal {
	return w.Balance.Sub(w.LockedBalance)
}

// CanDebit checks if the wallet has sufficient available balance for a debit
func (w *Wallet) CanDebit(amount decimal.Decimal) bool {
	return w.GetAvailableBalance().GreaterThanOrEqual(amount)
}

// LockFunds locks the specified amount in the wallet
func (w *Wallet) LockFunds(amount decimal.Decimal) error {
	if !w.CanDebit(amount) {
		return ErrInsufficientBalance
	}
	w.LockedBalance = w.LockedBalance.Add(amount)
	return nil
}

// UnlockFunds unlocks the specified amount in the wallet
func (w *Wallet) UnlockFunds(amount decimal.Decimal) error {
	if w.LockedBalance.LessThan(amount) {
		return ErrInvalidWalletBalance
	}
	w.LockedBalance = w.LockedBalance.Sub(amount)
	return nil
}

// Credit adds funds to the wallet
func (w *Wallet) Credit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	w.Balance = w.Balance.Add(amount)
	return nil
}

// IsOperationAllowed checks if wallet operations are allowed
func (w *Wallet) IsOperationAllowed() bool {
	return !w.IsLocked
}

// Lock locks the wallet with a reason
func (w *Wallet) Lock(reason string) {
	w.IsLocked = true
	w.LockReason = reason
}

// Unlock unlocks the wallet
func (w *Wallet) Unlock() {
	w.IsLocked = false
	w.LockReason = ""
}

// Debit removes funds from the wallet
func (w *Wallet) Debit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	if !w.CanDebit(amount) {
		return ErrInsufficientBalance
	}
	w.Balance = w.Balance.Sub(amount)
	return nil
}

// DebitLocked removes funds from locked balance and total balance
func (w *Wallet) DebitLocked(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidTransactionAmount
	}
	if w.LockedBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	w.LockedBalance = w.LockedBalance.Sub(amount)
	w.Balance = w.Balance.Sub(amount)
	return nil
}

// Validate performs validation on the wallet model
func (w *Wallet) Validate() error {
	if w.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if len(w.CurrencyCode) != 2 {
		return ErrInvalidCurrencyCode
	}
	if w.Balance.LessThan(decimal.Zero) {
		return ErrNegativeBalance
	}
	if w.LockedBalance.LessThan(decimal.Zero) {
		return ErrNegativeBalance
	}
	if w.LockedBalance.GreaterThan(w.Balance) {
		return ErrInvalidWalletBalance
	}
	return nil
}
