package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
)

// ExchangeHandler handles currency exchange rate requests
type ExchangeHandler struct{}

// NewExchangeHandler creates a new ExchangeHandler
func NewExchangeHandler() *ExchangeHandler {
	return &ExchangeHandler{}
}

// ExchangeRateResponse is the response for exchange rate queries
type ExchangeRateResponse struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
	Result float64 `json:"result"`
}

// cached rate entry
type cachedRate struct {
	rate      float64
	fetchedAt time.Time
}

var (
	rateCache   sync.Map
	rateCacheTTL = 24 * time.Hour
)

// frankfurterResponse is the response from api.frankfurter.app
type frankfurterResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Rates  map[string]float64 `json:"rates"`
}

// GetRate - GET /api/v1/exchange-rate?from=JPY&to=EUR&amount=1000
func (h *ExchangeHandler) GetRate(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	amountStr := c.DefaultQuery("amount", "1")

	if from == "" || to == "" {
		apierr.Fail(c, http.StatusBadRequest, "exchange_params_required", "Both currencies are required")
		return
	}

	if from == to {
		amount, _ := strconv.ParseFloat(amountStr, 64)
		c.JSON(http.StatusOK, ExchangeRateResponse{
			From: from, To: to, Rate: 1, Amount: amount, Result: amount,
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		apierr.Fail(c, http.StatusBadRequest, "invalid_amount", "Invalid amount")
		return
	}

	// Check cache
	cacheKey := from + "→" + to
	if cached, ok := rateCache.Load(cacheKey); ok {
		entry := cached.(cachedRate)
		if time.Since(entry.fetchedAt) < rateCacheTTL {
			result := amount * entry.rate
			c.JSON(http.StatusOK, ExchangeRateResponse{
				From: from, To: to, Rate: entry.rate, Amount: amount, Result: result,
			})
			return
		}
	}

	// Fetch from frankfurter.app
	url := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s&to=%s&amount=1", from, to)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		apierr.Fail(c, http.StatusServiceUnavailable, "exchange_rate_unavailable", "The exchange rate is unavailable. Enter it manually.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apierr.Fail(c, http.StatusServiceUnavailable, "exchange_rate_unavailable", "The exchange rate is unavailable. Enter it manually.")
		return
	}

	var fResp frankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&fResp); err != nil {
		apierr.Fail(c, http.StatusServiceUnavailable, "exchange_service_error", "The exchange-rate service returned an unusable response.")
		return
	}

	rate, ok := fResp.Rates[to]
	if !ok {
		apierr.FailWith(c, http.StatusBadRequest, "currency_not_supported",
			fmt.Sprintf("Currency %q is not supported", to),
			map[string]string{"currency": to})
		return
	}

	// Cache the rate
	rateCache.Store(cacheKey, cachedRate{rate: rate, fetchedAt: time.Now()})

	result := amount * rate
	c.JSON(http.StatusOK, ExchangeRateResponse{
		From: from, To: to, Rate: rate, Amount: amount, Result: result,
	})
}
