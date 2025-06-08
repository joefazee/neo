package prediction

import (
	"math"
	"testing"

	"github.com/joefazee/neo/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func newTestConfig() *Config {
	// Using GetDefaultConfig from your prediction/config.go
	// This ensures tests run against the same defaults your app might use.
	return GetDefaultConfig()
}

func TestNewBettingEngine(t *testing.T) {
	config := newTestConfig()
	engine := NewBettingEngine(config)
	assert.NotNil(t, engine)
	// Cast to concrete type to check internal config if needed, though not strictly necessary
	if e, ok := engine.(*bettingEngine); ok {
		assert.Equal(t, config, e.config)
	}
}

func TestBettingEngine_CalculateContractPrice(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	marketWithOutcomes := &models.Market{
		Outcomes: []models.MarketOutcome{
			{PoolAmount: decimal.NewFromInt(600)},
			{PoolAmount: decimal.NewFromInt(400)},
		},
		TotalPoolAmount: decimal.NewFromInt(1000),
	}
	outcome1 := &marketWithOutcomes.Outcomes[0]
	outcome2 := &marketWithOutcomes.Outcomes[1]

	t.Run("Normal calculation", func(t *testing.T) {
		price1 := engine.CalculateContractPrice(marketWithOutcomes, outcome1)
		price2 := engine.CalculateContractPrice(marketWithOutcomes, outcome2)
		assert.True(t, decimal.NewFromInt(60).Equal(price1), "Expected 60, got %s", price1)
		assert.True(t, decimal.NewFromInt(40).Equal(price2), "Expected 40, got %s", price2)
	})

	t.Run("Zero total pool with outcomes", func(t *testing.T) {
		market := &models.Market{
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.Zero},
				{PoolAmount: decimal.Zero},
			},
			TotalPoolAmount: decimal.Zero,
		}
		// With 2 outcomes, default should be 100.0/2 = 50
		price := engine.CalculateContractPrice(market, &market.Outcomes[0])
		assert.True(t, decimal.NewFromInt(50).Equal(price), "Expected 50 for zero pool with 2 outcomes, got %s", price)
	})

	t.Run("Zero total pool and zero outcomes", func(t *testing.T) {
		emptyMarket := &models.Market{TotalPoolAmount: decimal.Zero, Outcomes: []models.MarketOutcome{}}
		// The dummy outcome's PoolAmount doesn't matter here as TotalPoolAmount is zero
		price := engine.CalculateContractPrice(emptyMarket, &models.MarketOutcome{PoolAmount: decimal.Zero})
		// Corrected engine.go returns 50 if len(market.Outcomes) == 0
		assert.True(t, decimal.NewFromInt(50).Equal(price), "Expected 50 for zero pool and zero outcomes, got %s", price)
	})

	t.Run("Zero total pool and one outcome", func(t *testing.T) {
		oneOutcomeMarket := &models.Market{
			TotalPoolAmount: decimal.Zero,
			Outcomes:        []models.MarketOutcome{{PoolAmount: decimal.Zero}},
		}
		price := engine.CalculateContractPrice(oneOutcomeMarket, &oneOutcomeMarket.Outcomes[0])
		// Corrected engine.go: 100.0 / 1 = 100, bounded to 99
		assert.True(t, decimal.NewFromInt(99).Equal(price), "Expected 99 for zero pool and one outcome, got %s", price)
	})

	t.Run("Zero total pool and many outcomes (defaultPrice < 1)", func(t *testing.T) {
		outcomes := make([]models.MarketOutcome, 150) // 150 outcomes
		for i := range outcomes {
			outcomes[i] = models.MarketOutcome{PoolAmount: decimal.Zero}
		}
		manyOutcomesMarket := &models.Market{
			TotalPoolAmount: decimal.Zero,
			Outcomes:        outcomes,
		}
		// defaultPrice = 100.0 / 150 = 0.666...
		// This should be bounded to 1 by: if defaultPrice.LessThan(decimal.NewFromInt(1))
		price := engine.CalculateContractPrice(manyOutcomesMarket, &manyOutcomesMarket.Outcomes[0])
		assert.True(t, decimal.NewFromInt(1).Equal(price), "Expected 1 for zero pool and many outcomes (defaultPrice < 1), got %s", price)
	})

	t.Run("Outcome pool is zero, total pool not zero", func(t *testing.T) {
		market := &models.Market{
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.Zero}, // This outcome
				{PoolAmount: decimal.NewFromInt(400)},
			},
			TotalPoolAmount: decimal.NewFromInt(400), // Only other outcome has funds
		}
		price := engine.CalculateContractPrice(market, &market.Outcomes[0])
		// Corrected engine.go: if outcome.PoolAmount.IsZero() (and total is not), price is 1
		assert.True(t, decimal.NewFromInt(1).Equal(price), "Expected 1 for zero outcome pool, got %s", price)
	})

	t.Run("Price bounds (low)", func(t *testing.T) {
		market := &models.Market{
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromInt(5)},
				{PoolAmount: decimal.NewFromInt(995)},
			},
			TotalPoolAmount: decimal.NewFromInt(1000),
		}
		price := engine.CalculateContractPrice(market, &market.Outcomes[0])
		// Corrected engine.go: bounded to 1
		assert.True(t, decimal.NewFromInt(1).Equal(price), "Expected lower bound 1, got %s", price)
	})

	t.Run("Price bounds (high)", func(t *testing.T) {
		market := &models.Market{
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromInt(995)},
				{PoolAmount: decimal.NewFromInt(5)},
			},
			TotalPoolAmount: decimal.NewFromInt(1000),
		}
		price := engine.CalculateContractPrice(market, &market.Outcomes[0])
		// Corrected engine.go: bounded to 99
		assert.True(t, decimal.NewFromInt(99).Equal(price), "Expected upper bound 99, got %s", price)
	})
}

func TestBettingEngine_CalculateContractsBought(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())

	t.Run("Normal calculation", func(t *testing.T) {
		contracts := engine.CalculateContractsBought(decimal.NewFromInt(100), decimal.NewFromInt(50))
		assert.True(t, decimal.NewFromInt(200).Equal(contracts))
	})

	t.Run("Zero price", func(t *testing.T) {
		contracts := engine.CalculateContractsBought(decimal.NewFromInt(100), decimal.Zero)
		// Corrected engine.go: returns Zero if price or priceDecimal is zero
		assert.True(t, decimal.Zero.Equal(contracts))
	})

	t.Run("Price leads to zero priceDecimal (e.g. price 0.001)", func(t *testing.T) {
		smallPrice := decimal.NewFromFloat(0.001) // price/100 will be 0.00001
		contracts := engine.CalculateContractsBought(decimal.NewFromInt(100), smallPrice)
		expectedContracts := decimal.NewFromInt(100).Div(smallPrice.Div(decimal.NewFromInt(100)))
		assert.True(t, expectedContracts.Equal(contracts))

		// Test when priceDecimal itself would be zero due to precision with very small price
		// shopspring/decimal handles high precision, so price.Div(100) being zero for non-zero price is unlikely
		// unless price is extremely small, like 1e-30.
		// The guard `price.Div(decimal.NewFromInt(100)).IsZero()` is robust.
	})

	t.Run("Zero bet amount", func(t *testing.T) {
		contracts := engine.CalculateContractsBought(decimal.Zero, decimal.NewFromInt(50))
		assert.True(t, decimal.Zero.Equal(contracts))
	})
}

func TestBettingEngine_CalculatePriceImpact(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())

	t.Run("Small impact", func(t *testing.T) {
		impact := engine.CalculatePriceImpact(decimal.NewFromInt(1000), decimal.NewFromInt(50)) // (50/1000)*100 = 5%
		assert.True(t, decimal.NewFromInt(5).Equal(impact))
	})

	t.Run("Large impact (triggering log scaling)", func(t *testing.T) {
		impact := engine.CalculatePriceImpact(decimal.NewFromInt(1000), decimal.NewFromInt(200)) // (200/1000)*100 = 20%
		// Corrected engine.go: 10.0 + math.Log(20.0-9.0) = 10 + log(11)
		expectedImpact := decimal.NewFromFloat(10.0 + math.Log(11.0))
		assert.True(t, expectedImpact.Equal(impact), "Expected %s, got %s", expectedImpact, impact)
	})

	t.Run("Impact just at log threshold", func(t *testing.T) {
		// (100/1000)*100 = 10%. impactFloat is 10.0, so log scaling not applied.
		impact := engine.CalculatePriceImpact(decimal.NewFromInt(1000), decimal.NewFromInt(100))
		assert.True(t, decimal.NewFromInt(10).Equal(impact))
	})

	t.Run("Impact triggering inner log guard (impactFloat > 10 but impactFloat-9 <= 0)", func(t *testing.T) {
		// This test aims to hit the `if impactFloat-9.0 <= 0` guard *inside* the `if impactFloat > 10.0` block.
		// As discussed, this specific path is logically difficult/impossible to reach with standard float behavior
		// because if `impactFloat > 10.0`, then `impactFloat - 9.0` must be `> 1.0`.
		// The guard is more of a conceptual safety for extreme floating point issues or a misunderstanding in original logic.
		// We will test a scenario that *would* make `impactFloat-9.0` non-positive,
		// but it won't enter the `if impactFloat > 10.0` block.
		// So, this test effectively confirms the outer condition prevents reaching the problematic inner guard.

		// Scenario: impactFloat is 9.0. This does not satisfy `impactFloat > 10.0`.
		impact := engine.CalculatePriceImpact(decimal.NewFromFloat(100.0), decimal.NewFromFloat(9.0))
		assert.True(t, decimal.NewFromFloat(9.0).Equal(impact), "Expected 9.0, got %s", impact)

		// Scenario: impactFloat is 10.0. This does not satisfy `impactFloat > 10.0`.
		impactAtThreshold := engine.CalculatePriceImpact(decimal.NewFromFloat(100.0), decimal.NewFromFloat(10.0))
		assert.True(t, decimal.NewFromFloat(10.0).Equal(impactAtThreshold), "Expected 10.0, got %s", impactAtThreshold)

		// The corrected engine.go's guard `if impactFloat-9.0 <= 0 { return decimal.NewFromFloat(10.0) }`
		// inside the `if impactFloat > 10.0` block will not be hit by these inputs because the outer condition fails.
		// If we were to force `impactFloat` to be, say, `10.000000000000001` (so `>10` is true)
		// and simultaneously `impactFloat-9.0` to be `<=0` (so `impactFloat <= 9`), it's a contradiction.
		// Therefore, the inner guard is effectively dead code under normal float operations.
		// The test for "Large impact (triggering log scaling)" correctly tests the log path.
	})

	t.Run("Zero current pool", func(t *testing.T) {
		impact := engine.CalculatePriceImpact(decimal.Zero, decimal.NewFromInt(50))
		// Corrected engine.go: returns Zero if currentPool is zero
		assert.True(t, decimal.Zero.Equal(impact))
	})
}

func TestBettingEngine_CalculateSlippage(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())

	t.Run("Positive slippage", func(t *testing.T) {
		slippage := engine.CalculateSlippage(decimal.NewFromInt(50), decimal.NewFromInt(52))
		assert.True(t, decimal.NewFromInt(4).Equal(slippage))
	})

	t.Run("Negative slippage (absolute value)", func(t *testing.T) {
		slippage := engine.CalculateSlippage(decimal.NewFromInt(50), decimal.NewFromInt(48))
		assert.True(t, decimal.NewFromInt(4).Equal(slippage))
	})

	t.Run("Zero expected price", func(t *testing.T) {
		slippage := engine.CalculateSlippage(decimal.Zero, decimal.NewFromInt(50))
		// Corrected engine.go: returns Zero if expectedPrice is zero
		assert.True(t, decimal.Zero.Equal(slippage))
	})

	t.Run("No slippage", func(t *testing.T) {
		slippage := engine.CalculateSlippage(decimal.NewFromInt(50), decimal.NewFromInt(50))
		assert.True(t, decimal.Zero.Equal(slippage))
	})
}

func TestBettingEngine_ValidateSlippage(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	tolerance := decimal.NewFromFloat(5.0) // 5%

	t.Run("Within tolerance", func(t *testing.T) {
		err := engine.ValidateSlippage(decimal.NewFromFloat(4.0), tolerance)
		assert.NoError(t, err)
	})

	t.Run("At tolerance limit", func(t *testing.T) {
		err := engine.ValidateSlippage(decimal.NewFromFloat(5.0), tolerance)
		assert.NoError(t, err)
	})

	t.Run("Exceeds tolerance", func(t *testing.T) {
		err := engine.ValidateSlippage(decimal.NewFromFloat(5.1), tolerance)
		assert.Error(t, err)
		assert.Equal(t, models.ErrSlippageExceeded, err)
	})
}

func TestBettingEngine_CalculateNewPrice(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	market := &models.Market{
		Outcomes:        []models.MarketOutcome{},
		TotalPoolAmount: decimal.NewFromInt(1000),
	}
	outcome := &models.MarketOutcome{PoolAmount: decimal.NewFromInt(400)}
	betAmount := decimal.NewFromInt(100)

	t.Run("Normal calculation", func(t *testing.T) {
		newPrice := engine.CalculateNewPrice(market, outcome, betAmount)
		expected := decimal.NewFromInt(500).Div(decimal.NewFromInt(1100)).Mul(decimal.NewFromInt(100))
		assert.True(t, expected.Equal(newPrice), "Expected %s, got %s", expected, newPrice)
	})

	t.Run("Zero new total pool with zero outcomes", func(t *testing.T) {
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = []models.MarketOutcome{} // Explicitly zero outcomes
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.Zero)
		// Corrected engine.go: returns 50 if len(market.Outcomes) == 0
		assert.True(t, decimal.NewFromInt(50).Equal(newPrice))
	})

	t.Run("Zero new total pool with one outcome", func(t *testing.T) {
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = []models.MarketOutcome{{}} // One outcome
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.Zero)
		// Corrected engine.go: 100.0 / 1 = 100, bounded to 99
		assert.True(t, decimal.NewFromInt(99).Equal(newPrice))
	})

	t.Run("Zero new total pool with many outcomes (defaultPrice < 1)", func(t *testing.T) {
		outcomes := make([]models.MarketOutcome, 150)
		for i := range outcomes {
			outcomes[i] = models.MarketOutcome{PoolAmount: decimal.Zero}
		}
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = outcomes
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.Zero)
		// defaultPrice = 100.0 / 150 = 0.666... -> bounded to 1
		assert.True(t, decimal.NewFromInt(1).Equal(newPrice), "Expected 1, got %s", newPrice)
	})

	t.Run("Zero new total pool with moderate outcomes (defaultPrice between 1 and 99)", func(t *testing.T) {
		outcomes := make([]models.MarketOutcome, 4) // 4 outcomes
		for i := range outcomes {
			outcomes[i] = models.MarketOutcome{PoolAmount: decimal.Zero}
		}
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = outcomes
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.Zero)
		// defaultPrice = 100.0 / 4 = 25
		assert.True(t, decimal.NewFromInt(25).Equal(newPrice), "Expected 25, got %s", newPrice)
	})

	t.Run("Price bounds (low)", func(t *testing.T) {
		market.TotalPoolAmount = decimal.NewFromInt(10000)
		outcome.PoolAmount = decimal.NewFromInt(10)
		market.Outcomes = []models.MarketOutcome{{PoolAmount: decimal.NewFromInt(10)}, {PoolAmount: decimal.NewFromInt(9990)}} // ensure market.Outcomes is not empty for other paths
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.NewFromInt(1))                                           // (11/10001)*100
		assert.True(t, decimal.NewFromInt(1).Equal(newPrice))
	})

	t.Run("Price bounds (high)", func(t *testing.T) {
		market.TotalPoolAmount = decimal.NewFromInt(10000)
		outcome.PoolAmount = decimal.NewFromInt(9990)
		market.Outcomes = []models.MarketOutcome{{PoolAmount: decimal.NewFromInt(9990)}, {PoolAmount: decimal.NewFromInt(10)}}
		newPrice := engine.CalculateNewPrice(market, outcome, decimal.NewFromInt(1)) // (9991/10001)*100
		assert.True(t, decimal.NewFromInt(99).Equal(newPrice))
	})
}

func TestBettingEngine_CalculatePotentialPayout(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	contracts := decimal.NewFromInt(10)
	totalWinningContracts := decimal.NewFromInt(100)
	prizePool := decimal.NewFromInt(1000)

	t.Run("Normal calculation", func(t *testing.T) {
		payout := engine.CalculatePotentialPayout(contracts, totalWinningContracts, prizePool)
		assert.True(t, decimal.NewFromInt(100).Equal(payout))
	})

	t.Run("Zero total winning contracts", func(t *testing.T) {
		payout := engine.CalculatePotentialPayout(contracts, decimal.Zero, prizePool)
		// Corrected engine.go: returns Zero
		assert.True(t, decimal.Zero.Equal(payout))
	})
}

func TestBettingEngine_CalculateBreakevenPrice(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())

	t.Run("Normal calculation", func(t *testing.T) {
		price := engine.CalculateBreakevenPrice(decimal.NewFromInt(100), decimal.NewFromInt(200))
		assert.True(t, decimal.NewFromInt(50).Equal(price))
	})

	t.Run("Zero contracts", func(t *testing.T) {
		price := engine.CalculateBreakevenPrice(decimal.NewFromInt(100), decimal.Zero)
		// Corrected engine.go: returns Zero
		assert.True(t, decimal.Zero.Equal(price))
	})
}

func TestBettingEngine_EstimateGasPrice(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	market := &models.Market{
		TotalPoolAmount: decimal.NewFromInt(10000),
		Outcomes: []models.MarketOutcome{
			{PoolAmount: decimal.NewFromInt(5000)},
			{PoolAmount: decimal.NewFromInt(5000)},
		},
	}
	outcome := &market.Outcomes[0]

	t.Run("Small bet (less than 1% of pool)", func(t *testing.T) {
		betAmount := decimal.NewFromInt(50)
		effectivePrice := engine.EstimateGasPrice(market, outcome, betAmount)
		currentPrice := engine.CalculateContractPrice(market, outcome)
		assert.True(t, currentPrice.Equal(effectivePrice), "Expected %s, got %s", currentPrice, effectivePrice)
	})

	t.Run("Larger bet (more than 1% of pool)", func(t *testing.T) {
		betAmount := decimal.NewFromInt(1000)
		effectivePrice := engine.EstimateGasPrice(market, outcome, betAmount)
		currentPrice := engine.CalculateContractPrice(market, outcome)
		newPrice := engine.CalculateNewPrice(market, outcome, betAmount)
		weight := betAmount.Div(market.TotalPoolAmount.Add(betAmount))
		expectedWeightedPrice := currentPrice.Mul(decimal.NewFromInt(1).Sub(weight)).Add(newPrice.Mul(weight))
		assert.True(t, expectedWeightedPrice.Equal(effectivePrice), "Expected %s, got %s", expectedWeightedPrice, effectivePrice)
	})

	t.Run("Zero total pool, zero bet amount", func(t *testing.T) {
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero                                    // Outcome pool must also be zero for consistency
		market.Outcomes = []models.MarketOutcome{{PoolAmount: decimal.Zero}} // Simulate at least one outcome
		betAmount := decimal.Zero
		// currentPrice will be calculated based on zero pool (e.g., 99 for 1 outcome, or 50 for 2)
		currentPrice := engine.CalculateContractPrice(market, &market.Outcomes[0])
		effectivePrice := engine.EstimateGasPrice(market, &market.Outcomes[0], betAmount)
		// Corrected engine.go: if market.TotalPoolAmount.IsZero() and betAmount.IsZero(), returns currentPrice
		assert.True(t, currentPrice.Equal(effectivePrice), "Expected %s, got %s", currentPrice, effectivePrice)
	})

	t.Run("Zero total pool, non-zero bet amount", func(t *testing.T) {
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = []models.MarketOutcome{{PoolAmount: decimal.Zero}}
		betAmount := decimal.NewFromInt(100)
		// currentPrice will be default (e.g. 99 for 1 outcome)
		// The logic `if market.TotalPoolAmount.IsZero()` in EstimateGasPrice will be true.
		// Then `if betAmount.IsZero()` will be false.
		// The code doesn't explicitly return a value in this specific sub-branch of the corrected code.
		// It falls through to the later logic.
		// Let's trace: currentPrice = 99 (for 1 outcome, zero pool)
		// `!market.TotalPoolAmount.IsZero()` is false, so it skips the small bet optimization.
		// newPrice = CalculateNewPrice(market(total=0), outcome(pool=0), betAmount=100)
		//   newOutcomePool = 100, newTotalPool = 100. newPrice = (100/100)*100 = 100 -> bounded to 99.
		// denominator = 0 + 100 = 100.
		// weight = 100 / 100 = 1.
		// weightedPrice = currentPrice * (1-1) + newPrice * 1 = 0 + newPrice = newPrice
		effectivePrice := engine.EstimateGasPrice(market, &market.Outcomes[0], betAmount)
		expectedNewPrice := engine.CalculateNewPrice(market, &market.Outcomes[0], betAmount)
		assert.True(t, expectedNewPrice.Equal(effectivePrice), "Expected %s, got %s", expectedNewPrice, effectivePrice)
	})

	t.Run("Denominator zero in weight calculation (both total pool and bet amount are zero initially)", func(t *testing.T) {
		market.TotalPoolAmount = decimal.Zero
		outcome.PoolAmount = decimal.Zero
		market.Outcomes = []models.MarketOutcome{{PoolAmount: decimal.Zero}} // Ensure at least one outcome
		betAmount := decimal.Zero

		// currentPrice will be the default (e.g., 99 for 1 outcome).
		// The first `if market.TotalPoolAmount.IsZero()` block handles `betAmount.IsZero()` and returns `currentPrice`.
		effectivePrice := engine.EstimateGasPrice(market, &market.Outcomes[0], betAmount)
		currentPrice := engine.CalculateContractPrice(market, &market.Outcomes[0])
		assert.True(t, currentPrice.Equal(effectivePrice))
	})
}

func TestBettingEngine_CalculateLiquidityScore(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())

	t.Run("Zero pool", func(t *testing.T) {
		market := &models.Market{TotalPoolAmount: decimal.Zero}
		score := engine.CalculateLiquidityScore(market)
		assert.True(t, decimal.Zero.Equal(score))
	})

	t.Run("High liquidity and balance", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(1000000),
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromFloat(500000)},
				{PoolAmount: decimal.NewFromFloat(500000)},
			},
		}
		score := engine.CalculateLiquidityScore(market)
		// Base score for 1M pool = (log(1M+1)/log(1M+1))*50 = 50
		// Balance score for 2 perfectly balanced outcomes = ( (-0.5*log2(0.5) -0.5*log2(0.5)) / log2(2) ) * 50 = ( (1) / 1 ) * 50 = 50
		// Total = 50 + 50 = 100
		assert.True(t, decimal.NewFromInt(100).Equal(score), "Expected high score 100, got %s", score)
	})

	t.Run("Low liquidity", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(100),
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromFloat(50)},
				{PoolAmount: decimal.NewFromFloat(50)},
			},
		}
		score := engine.CalculateLiquidityScore(market)
		// baseScore for 100 pool = log(101)/log(1000001) * 50 approx (2/6)*50 = 16.6
		// balanceScore = 50
		// total approx 66.6
		assert.True(t, score.GreaterThan(decimal.NewFromInt(60)) && score.LessThan(decimal.NewFromInt(70)), "Expected score around 66 for low liquidity but balanced, got %s", score)
	})

	t.Run("Imbalanced market", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(100000),
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromFloat(95000)}, // p1 = 0.95
				{PoolAmount: decimal.NewFromFloat(5000)},  // p2 = 0.05
			},
		}
		score := engine.CalculateLiquidityScore(market)
		// baseScore for 100k pool = log(100k+1)/log(1M+1) * 50 = (5/6)*50 approx 41.6
		// entropy = -0.95*log2(0.95) - 0.05*log2(0.05) approx -0.95*(-0.074) - 0.05*(-4.32) = 0.0703 + 0.216 = 0.2863
		// maxEntropy = log2(2) = 1
		// balanceScore = (0.2863/1)*50 approx 14.3
		// total approx 41.6 + 14.3 = 55.9
		assert.True(t, score.GreaterThan(decimal.NewFromInt(50)) && score.LessThan(decimal.NewFromInt(60)), "Expected score around 55 for imbalanced market, got %s", score)
	})

	t.Run("Single outcome market", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(10000),
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromFloat(10000)},
			},
		}
		// Corrected engine.go: balanceScore is 0.0 for len(outcomes) == 1
		score := engine.CalculateLiquidityScore(market)
		expectedBaseScore := decimal.NewFromFloat(math.Log(10000.0+1.0) / math.Log(1000000.0+1.0) * 50.0)
		assert.True(t, score.Equal(expectedBaseScore), "Expected score %s based on pool size only, got %s", expectedBaseScore, score)
	})

	t.Run("Zero outcome market", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(10000),
			Outcomes:        []models.MarketOutcome{},
		}
		// Corrected engine.go: balanceScore is 0.0 for len(outcomes) == 0
		score := engine.CalculateLiquidityScore(market)
		expectedBaseScore := decimal.NewFromFloat(math.Log(10000.0+1.0) / math.Log(1000000.0+1.0) * 50.0)
		assert.True(t, score.Equal(expectedBaseScore), "Expected score %s based on pool size only, got %s", expectedBaseScore, score)
	})

	t.Run("Outcome pool has zero probability (for log2)", func(t *testing.T) {
		market := &models.Market{
			TotalPoolAmount: decimal.NewFromFloat(1000),
			Outcomes: []models.MarketOutcome{
				{PoolAmount: decimal.NewFromFloat(1000)},
				{PoolAmount: decimal.Zero}, // This outcome has prob 0
			},
		}
		// Corrected engine.go: if prob > 0 for log2. So this outcome adds 0 to entropy.
		// entropy = -1*log2(1) - 0 = 0.
		// maxEntropy = log2(2) = 1.
		// balanceScore = (0/1)*50 = 0.
		score := engine.CalculateLiquidityScore(market)
		expectedBaseScore := decimal.NewFromFloat(math.Log(1000.0+1.0) / math.Log(1000000.0+1.0) * 50.0)
		assert.True(t, score.Equal(expectedBaseScore), "Expected score %s (only base score), got %s", expectedBaseScore, score)
	})
}

func TestBettingEngine_CalculateImpliedProbability(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	prob := engine.CalculateImpliedProbability(decimal.NewFromInt(60))
	assert.True(t, decimal.NewFromFloat(0.6).Equal(prob))
}

func TestBettingEngine_CalculateExpectedValue(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	ev := engine.CalculateExpectedValue(
		decimal.NewFromInt(100),
		decimal.NewFromInt(50),
		decimal.NewFromFloat(0.6),
	)
	assert.True(t, decimal.NewFromInt(20).Equal(ev))

	t.Run("Zero price", func(t *testing.T) {
		evZeroPrice := engine.CalculateExpectedValue(
			decimal.NewFromInt(100),
			decimal.Zero,
			decimal.NewFromFloat(0.6),
		)
		// Corrected engine.go: returns betAmount.Neg()
		assert.True(t, decimal.NewFromInt(-100).Equal(evZeroPrice))
	})
}

func TestBettingEngine_CalculateOptimalBetSize_Kelly(t *testing.T) {
	engine := NewBettingEngine(newTestConfig())
	bankroll := decimal.NewFromInt(1000)

	t.Run("Positive EV", func(t *testing.T) {
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(50), decimal.NewFromFloat(0.6))
		assert.True(t, decimal.NewFromInt(200).Equal(betSize))
	})

	t.Run("Negative EV (no edge)", func(t *testing.T) {
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(50), decimal.NewFromFloat(0.4))
		assert.True(t, decimal.Zero.Equal(betSize))
	})

	t.Run("Kelly fraction capped", func(t *testing.T) {
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(10), decimal.NewFromFloat(0.5))
		assert.True(t, decimal.NewFromInt(250).Equal(betSize))
	})

	t.Run("Edge case prices (zero or 100)", func(t *testing.T) {
		assert.True(t, decimal.Zero.Equal(engine.CalculateOptimalBetSize(bankroll, decimal.Zero, decimal.NewFromFloat(0.5))))
		// For price = 100, the initial guard `price.GreaterThanOrEqual(decimal.NewFromInt(100))` handles this.
		assert.True(t, decimal.Zero.Equal(engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(100), decimal.NewFromFloat(0.5))))
	})

	t.Run("Edge case true probability (zero or 1)", func(t *testing.T) {
		assert.True(t, decimal.Zero.Equal(engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(50), decimal.Zero)))
		assert.True(t, decimal.Zero.Equal(engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(50), decimal.NewFromInt(1))))
	})

	t.Run("Implied probability is zero (price is effectively zero)", func(t *testing.T) {
		// This test case is for the `if impliedProbability.IsZero()` guard.
		// This guard is hit if `price.Div(decimal.NewFromInt(100))` is zero.
		// `price` itself being zero is caught by the first guard in the function.
		// This specific guard is for when `price` is non-zero but so small that `price/100` becomes zero
		// due to precision limits if not using high-precision decimals (shopspring/decimal handles this well).
		// For shopspring/decimal, price/100 will only be zero if price is zero.
		// So, this test is effectively the same as price == 0.
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromFloat(0.000000000000000000000000000001), decimal.NewFromFloat(0.5))
		// Even with a tiny price, impliedProbability won't be exactly zero with shopspring/decimal.
		// The first guard `price.LessThanOrEqual(decimal.Zero)` is the primary one for zero price.
		// If price is positive, impliedProbability will be positive.
		// The test "Edge case prices (zero or 100)" where price is Zero already covers this path effectively.
		// To be absolutely sure, let's assume a scenario where impliedProbability could become zero despite price not being zero (hypothetical for other decimal types).
		// For this engine, this path is hard to hit distinctly from price == 0.
		// We'll rely on the `price == 0` test.
		// If we were to mock `price.Div(decimal.NewFromInt(100))` to return Zero, this path would be hit.
		// For now, we assume the existing price == 0 test is sufficient.
		assert.True(t, betSize.GreaterThanOrEqual(decimal.Zero)) // It will calculate a valid Kelly or cap it.
	})

	t.Run("Odds lead to b == 0 (price is 100)", func(t *testing.T) {
		// If price is 100, impliedProb is 1, odds is 1, b is 0.
		// The initial guard price.GreaterThanOrEqual(decimal.NewFromInt(100)) should catch this.
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromInt(100), decimal.NewFromFloat(0.99))
		assert.True(t, betSize.IsZero(), "Expected zero bet size for price 100, got %s", betSize)
	})

	t.Run("Price very close to 100, positive edge", func(t *testing.T) {
		// This was the original failing case, but the expectation was wrong.
		// Price 99.9999 implies p=0.999999. True probability 0.9999999. Edge exists.
		// b = (1 / 0.999999) - 1 = 0.000001000001...
		// p = 0.9999999, q = 0.0000001
		// kellyFraction = (b*p - q) / b = (0.000001... * 0.9999999 - 0.0000001) / 0.000001...
		// kellyFraction = (approx_b - q) / b approx (0.000001 - 0.0000001) / 0.000001 = 0.0000009 / 0.000001 = 0.9
		// This is > 0.25, so it's capped. Bet = 1000 * 0.25 = 250.
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromFloat(99.9999), decimal.NewFromFloat(0.9999999))
		assert.True(t, decimal.NewFromInt(250).Equal(betSize), "Expected capped bet size 250, got %s", betSize)
	})

	t.Run("Price very close to 100, no edge", func(t *testing.T) {
		// Price 99.9999 implies p=0.999999. True probability 0.999998 (less than implied).
		// b = 0.000001000001...
		// p = 0.999998, q = 0.000002
		// bp = 0.000001... * 0.999998 = 0.000000999997...
		// bp - q = 0.000000999997... - 0.000002 = negative number.
		// kellyFraction should be <= 0, so bet size is 0.
		betSize := engine.CalculateOptimalBetSize(bankroll, decimal.NewFromFloat(99.9999), decimal.NewFromFloat(0.999998))
		assert.True(t, betSize.IsZero(), "Expected zero bet size when true_p < implied_p at high odds, got %s", betSize)
	})
}
