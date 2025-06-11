package wallet

import (
	"github.com/gin-gonic/gin"
	"github.com/joefazee/neo/internal/deps"
)

const (
	RepoKey    = "wallet_repository"
	ServiceKey = "wallet_service"
)

// MountAuthenticated mounts authenticated wallet routes
func MountAuthenticated(r *gin.RouterGroup, container *deps.Container) {
	handler := createHandler(container)

	// Wallet management
	walletsGroup := r.Group("/wallets")
	walletsGroup.POST("", handler.CreateWallet)
	walletsGroup.GET("/:id", handler.GetWallet)
	walletsGroup.GET("/:id/transactions", handler.GetWalletTransactions)

	// Wallet operations
	walletsGroup.POST("/:id/credit", handler.CreditWallet)
	walletsGroup.POST("/:id/debit", handler.DebitWallet)
	walletsGroup.POST("/:id/lock-funds", handler.LockFunds)
	walletsGroup.POST("/:id/unlock-funds", handler.UnlockFunds)
	walletsGroup.PATCH("/:id/lock", handler.LockWallet)

	// User wallets
	userGroup := r.Group("/users")
	userGroup.GET("/:user_id/wallets", handler.GetUserWallets)
}

// InitRepositories initializes and registers repositories and services for this module
func InitRepositories(container *deps.Container) {
	// Initialize repository
	repo := NewRepository(container.DB)
	container.RegisterRepository(RepoKey, repo)

	// Initialize service
	srv := NewService(repo, container.DB)
	container.RegisterService(ServiceKey, srv)
}

// createHandler creates a wallet handler with all dependencies
func createHandler(container *deps.Container) *Handler {
	srv := container.GetService(ServiceKey).(Service)
	return NewHandler(srv)
}
