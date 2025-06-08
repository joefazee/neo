package models

import "errors"

var (
	ErrInvalidCountryName    = errors.New("invalid country name")
	ErrInvalidCountryCode    = errors.New("invalid country code")
	ErrInvalidCurrencyCode   = errors.New("invalid currency code")
	ErrInvalidCurrencySymbol = errors.New("invalid currency symbol")
	ErrInvalidCountryID      = errors.New("invalid country ID")

	ErrInvalidCategoryName = errors.New("invalid category name")
	ErrInvalidCategorySlug = errors.New("invalid category slug")

	ErrInvalidEmail     = errors.New("invalid email address")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidUserID    = errors.New("invalid user ID")

	ErrInvalidMarketTitle    = errors.New("invalid market title")
	ErrInvalidMarketType     = errors.New("invalid market type")
	ErrInvalidMarketStatus   = errors.New("invalid market status")
	ErrInvalidMarketID       = errors.New("invalid market ID")
	ErrInvalidCloseTime      = errors.New("invalid close time")
	ErrInvalidResolutionTime = errors.New("invalid resolution deadline")
	ErrMarketAlreadyClosed   = errors.New("market is already closed")
	ErrMarketNotOpen         = errors.New("market is not open for betting")

	ErrInvalidOutcomeKey   = errors.New("invalid outcome key")
	ErrInvalidOutcomeLabel = errors.New("invalid outcome label")

	ErrInvalidBetAmount    = errors.New("invalid bet amount")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrBetTooLarge         = errors.New("bet amount exceeds maximum")
	ErrBetTooSmall         = errors.New("bet amount below minimum")
	ErrBetAlreadySettled   = errors.New("bet is already settled")

	ErrInvalidWalletBalance = errors.New("invalid wallet balance")
	ErrNegativeBalance      = errors.New("balance cannot be negative")

	ErrInvalidTransactionType   = errors.New("invalid transaction type")
	ErrInvalidTransactionAmount = errors.New("invalid transaction amount")

	ErrInvalidPaymentProvider   = errors.New("invalid payment provider")
	ErrInvalidProviderReference = errors.New("invalid provider reference")

	ErrInvalidAuditAction  = errors.New("invalid audit action")
	ErrInvalidResourceType = errors.New("invalid resource type")

	ErrInvalidTokenJTI     = errors.New("invalid token JTI")
	ErrTokenAlreadyExpired = errors.New("token already expired")

	ErrInvalidMarketRake               = errors.New("invalid market rake percentage")
	ErrInvalidCreatorRevenueShare      = errors.New("invalid creator revenue share")
	ErrInvalidMinQuorum                = errors.New("invalid minimum quorum amount")
	ErrInvalidHouseBotAmount           = errors.New("invalid house bot amount")
	ErrInvalidBetAmountLimits          = errors.New("invalid bet amount limits")
	ErrInvalidMarketDuration           = errors.New("invalid market duration")
	ErrInvalidMaxMarketsPerUser        = errors.New("invalid max markets per user")
	ErrDatabaseCredentialNotConfigured = errors.New("database credentials not configured")
	ErrInvalidPriceImpactThresholds    = errors.New("invalid price impact thresholds")
	ErrInvalidBetCancellationWindow    = errors.New("bet cancellation window cannot be negative")

	ErrInvalidSlippageLimit      = errors.New("invalid slippage limit")
	ErrInvalidPositionLimit      = errors.New("invalid position limit")
	ErrInvalidBetTimeout         = errors.New("invalid bet timeout")
	ErrInvalidRateLimit          = errors.New("invalid rate limit")
	ErrInvalidCooldownPeriod     = errors.New("invalid cooldown period")
	ErrSlippageExceeded          = errors.New("slippage tolerance exceeded")
	ErrPositionLimitExceeded     = errors.New("position limit exceeded")
	ErrMarketNotOpenForBetting   = errors.New("market not open for betting")
	ErrInsufficientWalletBalance = errors.New("insufficient wallet balance")
	ErrBetCooldownActive         = errors.New("bet cooldown period active")
	ErrDailyLimitExceeded        = errors.New("daily betting limit exceeded")
	ErrRateLimitExceeded         = errors.New("betting rate limit exceeded")

	ErrInvalidUUID    = errors.New("invalid UUID")
	ErrRecordNotFound = errors.New("record not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
)
