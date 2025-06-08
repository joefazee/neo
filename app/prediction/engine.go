package prediction

import (
	"math"

	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
)

// bettingEngine implements the BettingEngine interface
type bettingEngine struct {
	config *Config
}

// NewBettingEngine creates a new betting engine
func NewBettingEngine(config *Config) BettingEngine {
	return &bettingEngine{
		config: config,
	}
}

// CalculateContractPrice calculates the current price per contract for an outcome
// Price is calculated as a percentage (0-100) representing the market's assessment of probability
func (be *bettingEngine) CalculateContractPrice(market *models.Market, outcome *models.MarketOutcome) decimal.Decimal {
	if market.TotalPoolAmount.IsZero() {
		// Default price for new markets - equal probability
		if len(market.Outcomes) == 0 {
			return decimal.NewFromInt(50) // Default to 50 if no outcomes (should ideally not happen)
		}
		// Ensure price is bounded even with few outcomes
		defaultPrice := decimal.NewFromFloat(100.0 / float64(len(market.Outcomes)))
		if defaultPrice.LessThan(decimal.NewFromInt(1)) {
			return decimal.NewFromInt(1)
		}
		if defaultPrice.GreaterThan(decimal.NewFromInt(99)) {
			return decimal.NewFromInt(99)
		}
		return defaultPrice
	}

	if outcome.PoolAmount.IsZero() { // If this specific outcome has no pool, its price is effectively 1 (min)
		return decimal.NewFromInt(1)
	}

	// Price = (outcome pool / total pool) * 100
	outcomeRatio := outcome.PoolAmount.Div(market.TotalPoolAmount)
	price := outcomeRatio.Mul(decimal.NewFromInt(100))

	// Ensure price is within bounds [1, 99]
	if price.LessThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	if price.GreaterThan(decimal.NewFromInt(99)) {
		return decimal.NewFromInt(99)
	}

	return price
}

// CalculateContractsBought calculates how many contracts can be bought with the bet amount
func (be *bettingEngine) CalculateContractsBought(betAmount, price decimal.Decimal) decimal.Decimal {
	if price.IsZero() || price.Div(decimal.NewFromInt(100)).IsZero() { // Avoid division by zero for priceDecimal
		return decimal.Zero
	}

	// Contracts = bet amount / (price / 100)
	priceDecimal := price.Div(decimal.NewFromInt(100))
	return betAmount.Div(priceDecimal)
}

// CalculatePriceImpact calculates how much the price will move due to a bet
func (be *bettingEngine) CalculatePriceImpact(currentPool, betAmount decimal.Decimal) decimal.Decimal {
	if currentPool.IsZero() {
		return decimal.Zero // No impact if there's no existing pool to impact
	}

	// Price impact = (bet amount / current pool) * 100
	impact := betAmount.Div(currentPool).Mul(decimal.NewFromInt(100))

	// Apply diminishing returns for very large bets
	impactFloat := impact.InexactFloat64()
	if impactFloat > 10.0 {
		// Logarithmic scaling for large impacts
		adjustedImpact := 10.0 + math.Log(impactFloat-9.0) // Ensure impactFloat-9.0 > 0
		if impactFloat-9.0 <= 0 {                          // Guard against log of non-positive
			return decimal.NewFromFloat(10.0) // Cap at 10 if the argument to log becomes non-positive
		}
		impact = decimal.NewFromFloat(adjustedImpact)
	}

	return impact
}

// CalculateSlippage calculates the slippage between expected and actual price
func (be *bettingEngine) CalculateSlippage(expectedPrice, actualPrice decimal.Decimal) decimal.Decimal {
	if expectedPrice.IsZero() {
		return decimal.Zero // Avoid division by zero; if expected is 0, any actual price is infinite slippage, or treat as 0 for practical purposes.
	}

	// Slippage = |(actual - expected) / expected| * 100
	difference := actualPrice.Sub(expectedPrice).Abs()
	slippage := difference.Div(expectedPrice).Mul(decimal.NewFromInt(100))

	return slippage
}

// ValidateSlippage checks if slippage is within tolerance
func (be *bettingEngine) ValidateSlippage(slippage, tolerance decimal.Decimal) error {
	if slippage.GreaterThan(tolerance) {
		return models.ErrSlippageExceeded
	}
	return nil
}

// CalculateNewPrice calculates the new price after a bet is placed
func (be *bettingEngine) CalculateNewPrice(market *models.Market, outcome *models.MarketOutcome, betAmount decimal.Decimal) decimal.Decimal {
	// New outcome pool = current pool + bet amount
	newOutcomePool := outcome.PoolAmount.Add(betAmount)

	// New total pool = current total + bet amount
	newTotalPool := market.TotalPoolAmount.Add(betAmount)

	if newTotalPool.IsZero() {
		// If the new total pool is zero (e.g. refunding the only bet), default to 50%
		// or distribute among outcomes if any exist.
		if len(market.Outcomes) == 0 {
			return decimal.NewFromInt(50)
		}
		defaultPrice := decimal.NewFromFloat(100.0 / float64(len(market.Outcomes)))
		if defaultPrice.LessThan(decimal.NewFromInt(1)) {
			return decimal.NewFromInt(1)
		}
		if defaultPrice.GreaterThan(decimal.NewFromInt(99)) {
			return decimal.NewFromInt(99)
		}
		return defaultPrice
	}

	// New price = (new outcome pool / new total pool) * 100
	newPrice := newOutcomePool.Div(newTotalPool).Mul(decimal.NewFromInt(100))

	// Ensure price is within bounds
	if newPrice.LessThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	if newPrice.GreaterThan(decimal.NewFromInt(99)) {
		return decimal.NewFromInt(99)
	}

	return newPrice
}

// CalculatePotentialPayout calculates potential payout if the outcome wins
func (be *bettingEngine) CalculatePotentialPayout(contracts, totalWinningContracts, prizePool decimal.Decimal) decimal.Decimal {
	if totalWinningContracts.IsZero() {
		return decimal.Zero // Avoid division by zero
	}

	// Payout = (user's contracts / total winning contracts) * prize pool
	return contracts.Div(totalWinningContracts).Mul(prizePool)
}

// CalculateBreakevenPrice calculates the price at which the bet breaks even
// This assumes 'contracts' represents the number of shares/units that would pay out 100 currency units each if the price were 100.
func (be *bettingEngine) CalculateBreakevenPrice(betAmount, contracts decimal.Decimal) decimal.Decimal {
	if contracts.IsZero() {
		return decimal.Zero // Avoid division by zero
	}

	// Breakeven price = (bet amount / contracts) * 100
	// Example: Bet 100, got 200 contracts (implies price was 50). Breakeven is (100/200)*100 = 50.
	return betAmount.Div(contracts).Mul(decimal.NewFromInt(100))
}

// CalculateImpliedProbability converts price to implied probability
func (be *bettingEngine) CalculateImpliedProbability(price decimal.Decimal) decimal.Decimal {
	return price.Div(decimal.NewFromInt(100))
}

// CalculateExpectedValue calculates expected value of a bet
func (be *bettingEngine) CalculateExpectedValue(betAmount, price, trueProbability decimal.Decimal) decimal.Decimal {
	if price.IsZero() || price.Div(decimal.NewFromInt(100)).IsZero() { // Avoid division by zero for priceDecimal
		return betAmount.Neg() // If price is 0, payout is effectively infinite or undefined, EV is just -betAmount
	}
	// Expected payout if win
	priceDecimal := price.Div(decimal.NewFromInt(100))
	potentialPayout := betAmount.Div(priceDecimal)

	// Expected value = (probability * payout) - bet amount
	return trueProbability.Mul(potentialPayout).Sub(betAmount)
}

// CalculateOptimalBetSize calculates optimal bet size using Kelly criterion
func (be *bettingEngine) CalculateOptimalBetSize(bankroll, price, trueProbability decimal.Decimal) decimal.Decimal {
	if price.LessThanOrEqual(decimal.Zero) || price.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		return decimal.Zero
	}

	if trueProbability.LessThanOrEqual(decimal.Zero) || trueProbability.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		return decimal.Zero
	}

	// Convert price to decimal odds
	impliedProbability := price.Div(decimal.NewFromInt(100))
	if impliedProbability.IsZero() { // Avoid division by zero for odds
		return decimal.Zero
	}
	odds := decimal.NewFromInt(1).Div(impliedProbability)

	// Kelly formula: f = (bp - q) / b
	// where b = odds - 1, p = true probability, q = 1 - p
	b := odds.Sub(decimal.NewFromInt(1))
	p := trueProbability
	q := decimal.NewFromInt(1).Sub(p)

	if b.LessThanOrEqual(decimal.Zero) { // Odds must be greater than 1 (b > 0)
		return decimal.Zero
	}

	kellyFraction := b.Mul(p).Sub(q).Div(b)

	// Ensure non-negative and reasonable maximum (25% of bankroll)
	if kellyFraction.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	maxFraction := decimal.NewFromFloat(0.25) // Configurable?
	if kellyFraction.GreaterThan(maxFraction) {
		kellyFraction = maxFraction
	}

	return bankroll.Mul(kellyFraction)
}

// EstimateGasPrice estimates the effective price after accounting for market impact
func (be *bettingEngine) EstimateGasPrice(market *models.Market, outcome *models.MarketOutcome, betAmount decimal.Decimal) decimal.Decimal {
	currentPrice := be.CalculateContractPrice(market, outcome)
	if market.TotalPoolAmount.IsZero() { // If pool is zero, any bet makes it the new price.
		// Or, if betAmount is also zero, it remains the default price.
		if betAmount.IsZero() {
			return currentPrice // Which would be the default price (e.g., 50)
		}
		// If pool is zero but betAmount is not, the new price will be based on this single bet.
		// This case is complex as it implies the bet *is* the entire pool.
		// For simplicity, if initial pool is zero, let's assume impact makes it the new price directly.
		// This might need more sophisticated handling based on order book depth if that were a concept here.
		// A simpler approach: if total pool is zero, the first bet essentially sets the price.
		// The CalculateNewPrice logic handles this.
	}

	// For small bets, price impact is minimal
	// Check if TotalPoolAmount is zero before division
	if !market.TotalPoolAmount.IsZero() && betAmount.LessThan(market.TotalPoolAmount.Div(decimal.NewFromInt(100))) { // < 1% of pool
		return currentPrice
	}

	// For larger bets, calculate weighted average price
	newPrice := be.CalculateNewPrice(market, outcome, betAmount)

	// Weight calculation
	denominator := market.TotalPoolAmount.Add(betAmount)
	if denominator.IsZero() { // Avoid division by zero if both total pool and bet amount are zero
		return currentPrice // Or some default like 50
	}
	weight := betAmount.Div(denominator)
	weightedPrice := currentPrice.Mul(decimal.NewFromInt(1).Sub(weight)).Add(newPrice.Mul(weight))

	return weightedPrice
}

// CalculateLiquidityScore calculates a liquidity score for the market (0-100)
func (be *bettingEngine) CalculateLiquidityScore(market *models.Market) decimal.Decimal {
	totalPool := market.TotalPoolAmount.InexactFloat64()
	if totalPool <= 0 {
		return decimal.Zero
	}

	// Base score from pool size (logarithmic scale)
	// Max 50 points for pool size, scaled against a reference max pool (e.g., 1,000,000)
	referenceMaxPool := 1000000.0
	baseScore := math.Log(totalPool+1.0) / math.Log(referenceMaxPool+1.0) * 50.0
	baseScore = math.Min(baseScore, 50.0) // Cap at 50

	// Balance score - how evenly distributed the outcomes are
	balanceScore := 0.0
	if len(market.Outcomes) > 1 { // Entropy is undefined for 0 or 1 outcome
		entropy := 0.0
		for i := range market.Outcomes {
			outcome := market.Outcomes[i]
			if market.TotalPoolAmount.GreaterThan(decimal.Zero) {
				prob := outcome.PoolAmount.Div(market.TotalPoolAmount).InexactFloat64()
				if prob > 0 { // log2(0) is undefined
					entropy -= prob * math.Log2(prob)
				}
			}
		}

		maxEntropy := math.Log2(float64(len(market.Outcomes)))
		if maxEntropy > 0 { // Avoid division by zero if only one outcome (maxEntropy would be 0)
			balanceScore = (entropy / maxEntropy) * 50.0 // Max 50 points for balance
			balanceScore = math.Min(balanceScore, 50.0)  // Cap at 50
		}
	} else if len(market.Outcomes) == 1 {
		balanceScore = 0.0 // Perfectly imbalanced
	}

	totalScore := math.Min(baseScore+balanceScore, 100.0)
	return decimal.NewFromFloat(totalScore)
}
