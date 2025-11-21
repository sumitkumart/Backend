package price

import (
	"context"
	"hash/crc32"
	"math/rand"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Fetcher abstracts the external market data provider.
type Fetcher interface {
	Fetch(ctx context.Context, symbol string) (decimal.Decimal, error)
}

type RandomFetcher struct {
	min float64
	max float64

	mu  sync.Mutex
	rnd *rand.Rand
}

func NewRandomFetcher(min, max float64) *RandomFetcher {
	return &RandomFetcher{
		min: min,
		max: max,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (f *RandomFetcher) Fetch(_ context.Context, symbol string) (decimal.Decimal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	span := f.max - f.min
	value := f.min + f.rnd.Float64()*span
	offset := float64(crc32.ChecksumIEEE([]byte(symbol))%200) / 20.0
	value += offset
	return decimal.NewFromFloat(value).Round(2), nil
}
