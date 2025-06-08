package markets

import (
	"math"
)

// pricingEngine implements the PricingEngine interface
type pricingEngine struct {
	config *Config
}

// NewPricingEngine creates a new pricing engine
func NewPricingEngine(config *Config) PricingEngine {
	return &pricingEngine{
		config: config,
	}
}

// CalculatePrice calculates the current price of an outcome based on pool distribution
// Price is calculated as a percentage (0-100) representing the market's assessment of probability
func (pe *pricingEngine) CalculatePrice(totalPool, outcomePool float64) float64 {
	if totalPool <= 0 {
		return 50.0 // Default 50% if no betting activity
	}

	if outcomePool <= 0 {
		return 1.0 // Minimum price of 1%
	}

	// Calculate percentage of total pool
	percentage := (outcomePool / totalPool) * 100.0

	// Ensure price is within bounds [1, 99]
	if percentage < 1.0 {
		return 1.0
	}
	if percentage > 99.0 {
		return 99.0
	}

	return percentage
}

// CalculatePriceImpact calculates how much a bet will move the price
func (pe *pricingEngine) CalculatePriceImpact(currentPool, betAmount float64) float64 {
	if currentPool <= 0 {
		return 0.0
	}

	// Price impact is proportional to bet size relative to current pool
	impact := (betAmount / currentPool) * 100.0

	// Apply diminishing returns for very large bets
	if impact > 10.0 {
		impact = 10.0 + math.Log(impact-9.0)
	}

	return impact
}

// CalculateContractsBought calculates how many contracts a bet amount will buy at current price
func (pe *pricingEngine) CalculateContractsBought(betAmount, price float64) float64 {
	if price <= 0 {
		return 0.0
	}

	// Contracts = bet amount / (price / 100)
	// For example: ₦1000 bet at 50% price = 1000 / 0.50 = 2000 contracts
	priceDecimal := price / 100.0
	return betAmount / priceDecimal
}

// CalculatePayout calculates the payout for winning contracts
func (pe *pricingEngine) CalculatePayout(contracts, totalWinningContracts, prizePool float64) float64 {
	if totalWinningContracts <= 0 {
		return 0.0
	}

	// Payout = (user's contracts / total winning contracts) * prize pool
	return (contracts / totalWinningContracts) * prizePool
}

// CalculateImpliedProbability converts price to implied probability
func (pe *pricingEngine) CalculateImpliedProbability(price float64) float64 {
	return price / 100.0
}

// CalculatePriceFromProbability converts probability to price
func (pe *pricingEngine) CalculatePriceFromProbability(probability float64) float64 {
	price := probability * 100.0

	// Ensure price is within bounds
	if price < 1.0 {
		return 1.0
	}
	if price > 99.0 {
		return 99.0
	}

	return price
}

// CalculateExpectedValue calculates expected value of a bet
func (pe *pricingEngine) CalculateExpectedValue(betAmount, price, impliedProbability float64) float64 {
	// Expected payout if win
	potentialPayout := betAmount / (price / 100.0)

	// Expected value = (probability * payout) - bet amount
	return (impliedProbability * potentialPayout) - betAmount
}

// CalculateKellyBet calculates optimal bet size using Kelly criterion
func (pe *pricingEngine) CalculateKellyBet(bankroll, price, trueProbability float64) float64 {
	if price <= 0 || price >= 100 || trueProbability <= 0 || trueProbability >= 1 {
		return 0.0
	}

	// Convert price to decimal odds
	impliedProbability := price / 100.0
	odds := 1.0 / impliedProbability

	// Kelly formula: f = (bp - q) / b
	// where b = odds - 1, p = true probability, q = 1 - p
	b := odds - 1.0
	p := trueProbability
	q := 1.0 - p

	if b <= 0 {
		return 0.0
	}

	kellyFraction := (b*p - q) / b

	// Ensure non-negative and reasonable maximum
	if kellyFraction <= 0 {
		return 0.0
	}

	// Cap at 25% of bankroll for safety
	if kellyFraction > 0.25 {
		kellyFraction = 0.25
	}

	return bankroll * kellyFraction
}

// CalculateArbitrage reports whether an arbitrage exists and, if so, by how much.
func (pe *pricingEngine) CalculateArbitrage(prices []float64) (ok bool, margin float64) {
	if len(prices) < 2 {
		return
	}

	var totalProb float64
	for _, p := range prices {
		if p > 0 && p < 100 {
			totalProb += p / 100.0
		}
	}

	if totalProb < 1.0 {
		ok = true
		margin = 1.0 - totalProb
	}
	return
}

// CalculateVolatility calculates price volatility over time
func (pe *pricingEngine) CalculateVolatility(priceHistory []float64) float64 {
	if len(priceHistory) < 2 {
		return 0.0
	}

	// Calculate returns
	returns := make([]float64, len(priceHistory)-1)
	for i := 1; i < len(priceHistory); i++ {
		if priceHistory[i-1] > 0 {
			returns[i-1] = (priceHistory[i] - priceHistory[i-1]) / priceHistory[i-1]
		}
	}

	// Calculate mean return
	meanReturn := 0.0
	for _, ret := range returns {
		meanReturn += ret
	}
	meanReturn /= float64(len(returns))

	// Calculate variance
	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-meanReturn, 2)
	}
	variance /= float64(len(returns))

	// Return standard deviation (volatility)
	return math.Sqrt(variance)
}

// CalculateLiquidity calculates market liquidity score
func (pe *pricingEngine) CalculateLiquidity(totalPool, volume24h float64, uniqueBettors int) float64 {
	if totalPool <= 0 {
		return 0.0
	}

	// Base liquidity from pool size
	poolScore := math.Log(totalPool+1.0) / 10.0

	// Volume component
	volumeScore := 0.0
	if totalPool > 0 {
		volumeRatio := volume24h / totalPool
		volumeScore = math.Min(volumeRatio, 1.0)
	}

	// Diversity component (number of unique bettors)
	diversityScore := math.Min(float64(uniqueBettors)/50.0, 1.0)

	// Weighted combination
	liquidityScore := poolScore*0.5 + volumeScore*0.3 + diversityScore*0.2

	// Normalize to 0-100 scale
	return math.Min(liquidityScore*100.0, 100.0)
}

// CalculateSpread calculates bid-ask spread approximation
func (pe *pricingEngine) CalculateSpread(price, liquidity float64) float64 {
	if liquidity <= 0 {
		return 10.0 // High spread for illiquid markets
	}

	// Base spread inversely related to liquidity
	baseSpread := 100.0 / liquidity

	// Spread wider at extreme prices
	if price < 10.0 || price > 90.0 {
		baseSpread *= 1.5
	}

	// Minimum spread of 0.5%, maximum of 10%
	spread := math.Max(0.5, math.Min(baseSpread, 10.0))

	return spread
}
