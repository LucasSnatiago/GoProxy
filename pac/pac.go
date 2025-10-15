package pac

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/LucasSnatiago/gopac"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/net/proxy"
)

type Pac struct {
	PacCache      *expirable.LRU[string, string] // Cache for PAC entries
	Auth          *proxy.Auth                    // Optional authentication for the PAC script
	pacScript     string                         // The PAC script content
	ttlDuration   time.Duration                  // Duration for which the PAC entries are cached
	*sync.RWMutex                                // Mutex to protect access to the pool
	*sync.Pool                                   // Pool of gopac.Parser instances
}

func NewPac(pacScript string, ttl time.Duration) (*Pac, error) {
	vmPool := sync.Pool{
		New: func() any {
			vm := new(gopac.Parser)
			if err := vm.ParseBytes([]byte(pacScript)); err != nil {
				return fmt.Errorf("failed to load PAC script: %v", err)
			}
			return vm
		},
	}

	// Start metrics for cache hits and misses
	cacheHits = 0
	cacheMisses = 0
	startCacheStatsLogger()
	return &Pac{
		PacCache:    expirable.NewLRU[string, string](1000, nil, ttl*100), // Caching a thousand most recent visited sites
		Auth:        nil,                                                  // No authentication by default
		pacScript:   pacScript,
		ttlDuration: ttl,
		Pool:        &vmPool,
	}, nil
}

func (pac *Pac) Reload() error {
	newPac, err := NewPac(pac.pacScript, pac.ttlDuration)
	if err != nil {
		return fmt.Errorf("failed to reload PAC: %v", err)
	}

	pac.PacCache = newPac.PacCache
	pac.Auth = newPac.Auth
	pac.ttlDuration = newPac.ttlDuration
	pac.Pool = newPac.Pool
	return nil
}

func DownloadPAC(pacURL string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 300,
	}

	resp, err := client.Get(pacURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download pac from %s: %w", pacURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read answer from PAC: %w", err)
	}

	return string(data), nil
}

func (pac *Pac) SetAuth(username, password string) {
	if username != "" && password != "" {
		pac.Auth = &proxy.Auth{
			User:     username,
			Password: password,
		}
	}
}

func (p *Pac) PacCacheToString() (string, error) {
	keys := append([]string(nil), p.PacCache.Keys()...)
	sort.Strings(keys)

	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := p.PacCache.Get(k); ok {
			out[k] = v
		}
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
