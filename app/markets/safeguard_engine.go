package markets

import (
	"math"

	"github.com/joefazee/neo/models"
)

// safeguardEngine implements the SafeguardEngine interface
type safeguardEngine struct {
	config *Config
}

// NewSafeguardEngine creates a new safeguard engine
func NewSafeguardEngine(config *Config) SafeguardEngine {
	return &safeguardEngine{
		config: config,
	}
}

// CheckQuorum verifies if the market meets minimum quorum requirements
func (se *safeguardEngine) CheckQuorum(market *models.Market, outcomes []models.MarketOutcome) bool {
	// Check total pool amount
	if market.TotalPoolAmount.LessThan(market.SafeguardConfig.MinQuorumAmount) {
		return false
	}

	// Check minimum number of outcomes with bets
	outcomesWithBets := 0
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.PoolAmount.GreaterThan(se.config.MinBetAmount) {
			outcomesWithBets++
		}
	}

	return outcomesWithBets >= market.SafeguardConfig.MinOutcomes
}

// CheckImbalance checks if the market is severely imbalanced
func (se *safeguardEngine) CheckImbalance(outcomes []models.MarketOutcome, threshold float64) bool {
	if len(outcomes) < 2 {
		return true // Single outcome is always imbalanced
	}

	// Calculate total pool
	totalPool := 0.0
	maxPool := 0.0

	for i := range outcomes {
		outcome := outcomes[i]
		poolAmount := outcome.PoolAmount.InexactFloat64()
		totalPool += poolAmount
		if poolAmount > maxPool {
			maxPool = poolAmount
		}
	}

	if totalPool <= 0 {
		return false // No imbalance if no bets
	}

	// Check if any single outcome has more than threshold of total pool
	maxRatio := maxPool / totalPool
	return maxRatio <= threshold
}

// ShouldTriggerHouseBot determines if house bot should intervene
func (se *safeguardEngine) ShouldTriggerHouseBot(market *models.Market, outcomes []models.MarketOutcome) bool {
	if !market.SafeguardConfig.HouseBotEnabled || !se.config.EnableHouseBot {
		return false
	}

	// Don't trigger if market already has sufficient liquidity
	if market.TotalPoolAmount.GreaterThan(market.SafeguardConfig.HouseBotAmount.Mul(se.config.MinBetAmount)) {
		return false
	}

	// Check for severe imbalance
	if !se.CheckImbalance(outcomes, 0.9) { // 90% threshold for house bot
		return true
	}

	// Check if only one outcome has bets
	outcomesWithBets := 0
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.PoolAmount.GreaterThan(se.config.MinBetAmount) {
			outcomesWithBets++
		}
	}

	return outcomesWithBets < 2
}

// CalculateHouseBotPosition calculates optimal house bot betting position
func (se *safeguardEngine) CalculateHouseBotPosition(market *models.Market, outcomes []models.MarketOutcome) map[string]float64 {
	positions := make(map[string]float64)

	if !se.ShouldTriggerHouseBot(market, outcomes) {
		return positions
	}

	totalBudget := market.SafeguardConfig.HouseBotAmount.InexactFloat64()

	// Strategy 1: Balance extremely imbalanced markets
	if se.isExtremelyImbalanced(outcomes) {
		return se.calculateBalancingPosition(outcomes, totalBudget)
	}

	// Strategy 2: Provide liquidity to empty outcomes
	emptyOutcomes := se.findEmptyOutcomes(outcomes)
	if len(emptyOutcomes) > 0 {
		return se.calculateLiquidityPosition(emptyOutcomes, totalBudget)
	}

	// Strategy 3: Market making - bet against extreme prices
	return se.calculateMarketMakingPosition(outcomes, totalBudget)
}

// CheckVoidRisk determines whether a market should be voided.
func (se *safeguardEngine) CheckVoidRisk(
	market *models.Market,
	outcomes []models.MarketOutcome,
) (shouldVoid bool, reason string) {
	// 1) Quorum failure close to deadline
	if !se.CheckQuorum(market, outcomes) {
		hoursToClose := market.CloseTime.Sub(market.CreatedAt).Hours()
		if hoursToClose < 24 && market.SafeguardConfig.VoidOnQuorumFail {
			return true, "market failed to meet minimum quorum requirements"
		}
	}

	// 2) Manipulation patterns
	if se.detectManipulation(outcomes) {
		return true, "potential market manipulation detected"
	}

	// 3) Technical issues
	if se.hasTechnicalIssues(market, outcomes) {
		return true, "technical issues affecting market integrity"
	}

	return false, ""
}

// CalculateRiskScore calculates overall market risk score (0-100)
func (se *safeguardEngine) CalculateRiskScore(market *models.Market, outcomes []models.MarketOutcome) float64 {
	score := 0.0

	// Quorum risk (25 points)
	if !se.CheckQuorum(market, outcomes) {
		score += 25.0
	}

	// Imbalance risk (20 points)
	if !se.CheckImbalance(outcomes, 0.8) {
		score += 20.0
	}

	// Liquidity risk (15 points)
	liquidityScore := se.calculateLiquidityRisk(market)
	score += liquidityScore * 15.0

	// Time risk (10 points)
	timeRisk := se.calculateTimeRisk(market)
	score += timeRisk * 10.0

	// Manipulation risk (20 points)
	if se.detectManipulation(outcomes) {
		score += 20.0
	}

	// Volatility risk (10 points)
	volatilityRisk := se.calculateVolatilityRisk(outcomes)
	score += volatilityRisk * 10.0

	return math.Min(score, 100.0)
}

// Helper methods

func (se *safeguardEngine) isExtremelyImbalanced(outcomes []models.MarketOutcome) bool {
	return !se.CheckImbalance(outcomes, 0.95) // 95% threshold for extreme imbalance
}

func (se *safeguardEngine) findEmptyOutcomes(outcomes []models.MarketOutcome) []models.MarketOutcome {
	var empty []models.MarketOutcome
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.PoolAmount.LessThanOrEqual(se.config.MinBetAmount) {
			empty = append(empty, outcome)
		}
	}
	return empty
}

func (se *safeguardEngine) calculateBalancingPosition(outcomes []models.MarketOutcome, budget float64) map[string]float64 {
	positions := make(map[string]float64)

	// Find the outcome with least betting activity
	var minOutcome *models.MarketOutcome
	minAmount := math.MaxFloat64

	for i := range outcomes {
		outcome := outcomes[i]
		amount := outcome.PoolAmount.InexactFloat64()
		if amount < minAmount {
			minAmount = amount
			minOutcome = &outcomes[i]
		}
	}

	if minOutcome != nil {
		// Bet entire budget on least popular outcome
		positions[minOutcome.OutcomeKey] = budget
	}

	return positions
}

func (se *safeguardEngine) calculateLiquidityPosition(emptyOutcomes []models.MarketOutcome, budget float64) map[string]float64 {
	positions := make(map[string]float64)

	if len(emptyOutcomes) == 0 {
		return positions
	}

	// Distribute budget equally among empty outcomes
	perOutcome := budget / float64(len(emptyOutcomes))

	for i := range emptyOutcomes {
		outcome := emptyOutcomes[i]
		positions[outcome.OutcomeKey] = perOutcome
	}

	return positions
}

func (se *safeguardEngine) calculateMarketMakingPosition(outcomes []models.MarketOutcome, budget float64) map[string]float64 {
	positions := make(map[string]float64)

	// Calculate total pool
	totalPool := 0.0
	for i := range outcomes {
		outcome := outcomes[i]
		totalPool += outcome.PoolAmount.InexactFloat64()
	}

	if totalPool <= 0 {
		return positions
	}

	// Find outcomes with extreme prices (< 10% or > 90%)
	var extremeOutcomes []models.MarketOutcome

	for i := range outcomes {
		outcome := outcomes[i]
		price := (outcome.PoolAmount.InexactFloat64() / totalPool) * 100.0
		if price < 10.0 || price > 90.0 {
			extremeOutcomes = append(extremeOutcomes, outcome)
		}
	}

	if len(extremeOutcomes) == 0 {
		return positions
	}

	// Bet against extreme prices
	perOutcome := budget / float64(len(extremeOutcomes))

	for i := range extremeOutcomes {
		outcome := extremeOutcomes[i]
		price := (outcome.PoolAmount.InexactFloat64() / totalPool) * 100.0
		if price < 10.0 {
			// Bet on very cheap outcomes
			positions[outcome.OutcomeKey] = perOutcome
		}
		// Don't bet on very expensive outcomes (> 90%)
	}

	return positions
}

func (se *safeguardEngine) detectManipulation(outcomes []models.MarketOutcome) bool {
	// Check for suspicious betting patterns
	// This is a simplified implementation - real detection would be more sophisticated

	// Check for single large bet dominating outcome
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.PoolAmount.GreaterThan(se.config.MaxBetAmount.Mul(se.config.MinBetAmount)) {
			// Very large position - potential manipulation
			return true
		}
	}

	return false
}

func (se *safeguardEngine) hasTechnicalIssues(market *models.Market, outcomes []models.MarketOutcome) bool {
	// Check for technical problems that might affect market integrity

	// Check for data inconsistencies
	calculatedTotal := 0.0
	for i := range outcomes {
		outcome := outcomes[i]
		calculatedTotal += outcome.PoolAmount.InexactFloat64()
	}

	actualTotal := market.TotalPoolAmount.InexactFloat64()

	// Allow for small rounding differences
	if math.Abs(calculatedTotal-actualTotal) > 0.01 {
		return true
	}

	return false
}

func (se *safeguardEngine) calculateLiquidityRisk(market *models.Market) float64 {
	// Risk increases as liquidity decreases
	totalPool := market.TotalPoolAmount.InexactFloat64()
	optimalPool := se.config.MinQuorumAmount.InexactFloat64() * 5.0 // 5x minimum as optimal

	if totalPool >= optimalPool {
		return 0.0
	}

	return 1.0 - (totalPool / optimalPool)
}

func (se *safeguardEngine) calculateTimeRisk(market *models.Market) float64 {
	// Risk increases as market approaches close time without sufficient activity
	timeToClose := market.CloseTime.Sub(market.CreatedAt).Hours()
	totalTime := market.CloseTime.Sub(market.CreatedAt).Hours()

	if totalTime <= 0 {
		return 1.0
	}

	timeElapsed := 1.0 - (timeToClose / totalTime)

	// Risk increases exponentially in final 25% of time
	if timeElapsed > 0.75 {
		return math.Pow((timeElapsed-0.75)/0.25, 2)
	}

	return 0.0
}

func (se *safeguardEngine) calculateVolatilityRisk(outcomes []models.MarketOutcome) float64 {
	if len(outcomes) < 2 {
		return 1.0
	}

	var totalPool float64
	for i := range outcomes {
		totalPool += outcomes[i].PoolAmount.InexactFloat64()
	}
	if totalPool <= 0 {
		return 0.0
	}

	prices := make([]float64, 0, len(outcomes))
	for i := range outcomes {
		price := (outcomes[i].PoolAmount.InexactFloat64() / totalPool) * 100.0
		prices = append(prices, price)
	}

	mean := 100.0 / float64(len(prices))
	var variance float64
	for _, p := range prices {
		d := p - mean
		variance += d * d
	}
	variance /= float64(len(prices))

	return math.Min(math.Sqrt(variance)/50.0, 1.0)
}
