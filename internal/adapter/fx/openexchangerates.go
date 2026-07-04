package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// OpenExchangeRatesProvider fetches rates from the OpenExchangeRates API.
// Free tier provides USD-based rates. Cross-rates are calculated via USD.
type OpenExchangeRatesProvider struct {
	appID   string
	baseURL string

	mu        sync.RWMutex
	rates     map[string]float64 // currency -> rate relative to USD
	fetchedAt time.Time
	cacheTTL  time.Duration
}

func NewOpenExchangeRatesProvider(appID string) *OpenExchangeRatesProvider {
	return &OpenExchangeRatesProvider{
		appID:    appID,
		baseURL:  "https://openexchangerates.org/api",
		rates:    make(map[string]float64),
		cacheTTL: 1 * time.Hour,
	}
}

type oxrResponse struct {
	Timestamp int64              `json:"timestamp"`
	Base      string             `json:"base"`
	Rates     map[string]float64 `json:"rates"`
}

func (p *OpenExchangeRatesProvider) fetchRates(ctx context.Context) error {
	p.mu.RLock()
	if time.Since(p.fetchedAt) < p.cacheTTL && len(p.rates) > 0 {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	url := fmt.Sprintf("%s/latest.json?app_id=%s", p.baseURL, p.appID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create OXR request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("OXR API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("OXR API error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read OXR response: %w", err)
	}

	var oxr oxrResponse
	if err := json.Unmarshal(body, &oxr); err != nil {
		return fmt.Errorf("failed to parse OXR response: %w", err)
	}

	p.mu.Lock()
	p.rates = oxr.Rates
	p.rates["USD"] = 1.0
	p.fetchedAt = time.Now()
	p.mu.Unlock()

	return nil
}

func (p *OpenExchangeRatesProvider) GetRate(ctx context.Context, from, to string) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	if err := p.fetchRates(ctx); err != nil {
		return 0, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	fromRate, okFrom := p.rates[from]
	toRate, okTo := p.rates[to]
	if !okFrom {
		return 0, fmt.Errorf("currency not supported: %s", from)
	}
	if !okTo {
		return 0, fmt.Errorf("currency not supported: %s", to)
	}

	// Cross-rate: FROM -> USD -> TO
	// 1 FROM = (1/fromRate) USD = (toRate/fromRate) TO
	return toRate / fromRate, nil
}

func (p *OpenExchangeRatesProvider) Convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) {
	rate, err := p.GetRate(ctx, from, to)
	if err != nil {
		return 0, 0, err
	}
	converted := int64(float64(amount) * rate)
	return converted, rate, nil
}

func (p *OpenExchangeRatesProvider) ListRates(ctx context.Context, baseCurrency string) ([]port.ExchangeRate, error) {
	if err := p.fetchRates(ctx); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	baseRate, ok := p.rates[baseCurrency]
	if !ok {
		return nil, fmt.Errorf("base currency not supported: %s", baseCurrency)
	}

	var rates []port.ExchangeRate
	for currency, rate := range p.rates {
		if currency == baseCurrency {
			continue
		}
		rates = append(rates, port.ExchangeRate{
			From: baseCurrency,
			To:   currency,
			Rate: rate / baseRate,
		})
	}

	return rates, nil
}
