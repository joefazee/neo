package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
	TransactionTypeBetPlace   TransactionType = "bet_place"
	TransactionTypeBetRefund  TransactionType = "bet_refund"
	TransactionTypePayout     TransactionType = "payout"
	TransactionTypeFee        TransactionType = "fee"
)

// TransactionMetadata represents additional transaction metadata
type TransactionMetadata struct {
	IPAddress       string          `json:"ip_address,omitempty"`
	UserAgent       string          `json:"user_agent,omitempty"`
	PaymentProvider string          `json:"payment_provider,omitempty"`
	ExchangeRate    decimal.Decimal `json:"exchange_rate,omitempty"`
	FeeAmount       decimal.Decimal `json:"fee_amount,omitempty"`
	Notes           string          `json:"notes,omitempty"`
}

// Value implements driver.Valuer interface
func (tm *TransactionMetadata) Value() (driver.Value, error) {
	return json.Marshal(tm)
}

// Scan implements sql.Scanner interface
func (tm *TransactionMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, tm)
	case string:
		return json.Unmarshal([]byte(v), tm)
	}
	return nil
}

// Transaction represents a financial transaction (immutable ledger)
type Transaction struct {
	ID              uuid.UUID           `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID           `gorm:"type:uuid;not null;index:idx_transactions_user" json:"user_id"`
	WalletID        uuid.UUID           `gorm:"type:uuid;not null" json:"wallet_id"`
	TransactionType TransactionType     `gorm:"type:varchar(20);not null" json:"transaction_type"`
	Amount          decimal.Decimal     `gorm:"type:decimal(20,2);not null" json:"amount"`
	BalanceBefore   decimal.Decimal     `gorm:"type:decimal(20,2);not null" json:"balance_before"`
	BalanceAfter    decimal.Decimal     `gorm:"type:decimal(20,2);not null" json:"balance_after"`
	ReferenceType   string              `gorm:"type:varchar(20)" json:"reference_type"` // 'bet', 'settlement', 'payment'
	ReferenceID     *uuid.UUID          `gorm:"type:uuid" json:"reference_id"`
	Description     string              `gorm:"type:text" json:"description"`
	Metadata        TransactionMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt       time.Time           `gorm:"autoCreateTime;index:idx_transactions_created_at" json:"created_at"`

	// Associations (Note: Transactions are immutable, no updates)
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Wallet *Wallet `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
}

// TableName specifies the table name for Transaction model
func (*Transaction) TableName() string {
	return "transactions"
}

// BeforeCreate sets up the model before creation
func (t *Transaction) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// IsCredit checks if this is a credit transaction (positive amount)
func (t *Transaction) IsCredit() bool {
	return t.Amount.GreaterThan(decimal.Zero)
}

// IsDebit checks if this is a debit transaction (negative amount)
func (t *Transaction) IsDebit() bool {
	return t.Amount.LessThan(decimal.Zero)
}

// GetAbsoluteAmount returns the absolute value of the transaction amount
func (t *Transaction) GetAbsoluteAmount() decimal.Decimal {
	return t.Amount.Abs()
}

// IsBalanceConsistent checks if the balance calculation is consistent
func (t *Transaction) IsBalanceConsistent() bool {
	expectedBalance := t.BalanceBefore.Add(t.Amount)
	return expectedBalance.Equal(t.BalanceAfter)
}

// Validate performs validation on the transaction model
func (t *Transaction) Validate() error {
	if t.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if t.WalletID == uuid.Nil {
		return ErrInvalidWalletBalance
	}
	if t.Amount.IsZero() {
		return ErrInvalidTransactionAmount
	}
	if !t.IsBalanceConsistent() {
		return ErrInvalidTransactionAmount
	}
	if t.BalanceAfter.LessThan(decimal.Zero) {
		return ErrNegativeBalance
	}
	return nil
}

// CreateDepositTransaction creates a deposit transaction
func CreateDepositTransaction(userID,
	walletID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	paymentRef string) *Transaction {
	return &Transaction{
		UserID:          userID,
		WalletID:        walletID,
		TransactionType: TransactionTypeDeposit,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore.Add(amount),
		ReferenceType:   "payment",
		ReferenceID:     parseUUIDPtr(paymentRef),
		Description:     "Deposit to wallet",
	}
}

// CreateWithdrawalTransaction creates a withdrawal transaction
func CreateWithdrawalTransaction(userID,
	walletID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	paymentRef string) *Transaction {
	return &Transaction{
		UserID:          userID,
		WalletID:        walletID,
		TransactionType: TransactionTypeWithdrawal,
		Amount:          amount.Neg(), // Negative for withdrawal
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore.Sub(amount),
		ReferenceType:   "payment",
		ReferenceID:     parseUUIDPtr(paymentRef),
		Description:     "Withdrawal from wallet",
	}
}

// CreateBetTransaction creates a bet placement transaction
func CreateBetTransaction(userID,
	walletID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	betID uuid.UUID) *Transaction {
	return &Transaction{
		UserID:          userID,
		WalletID:        walletID,
		TransactionType: TransactionTypeBetPlace,
		Amount:          amount.Neg(), // Negative for bet placement
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore.Sub(amount),
		ReferenceType:   "bet",
		ReferenceID:     &betID,
		Description:     "Bet placement",
	}
}

// CreatePayoutTransaction creates a payout transaction
func CreatePayoutTransaction(userID,
	walletID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	settlementID uuid.UUID) *Transaction {
	return &Transaction{
		UserID:          userID,
		WalletID:        walletID,
		TransactionType: TransactionTypePayout,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore.Add(amount),
		ReferenceType:   "settlement",
		ReferenceID:     &settlementID,
		Description:     "Bet payout",
	}
}

// CreateBetRefundTransaction creates a bet refund transaction
func CreateBetRefundTransaction(userID,
	walletID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	betID uuid.UUID) *Transaction {
	return &Transaction{
		UserID:          userID,
		WalletID:        walletID,
		TransactionType: TransactionTypeBetRefund,
		Amount:          amount, // Positive for refund (credit)
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore.Add(amount),
		ReferenceType:   "bet",
		ReferenceID:     &betID,
		Description:     "Refund for canceled bet",
	}
}

// parseUUIDPtr safely parses a string to UUID pointer
func parseUUIDPtr(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	if id, err := uuid.Parse(s); err == nil {
		return &id
	}
	return nil
}
