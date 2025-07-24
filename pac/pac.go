package pac

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackwakefield/gopac"
)

type Pac struct {
	PacCache *expirable.LRU[string, string]
	*sync.Pool
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

	return &Pac{
		PacCache: expirable.NewLRU[string, string](1000000, nil, ttl), // Caching the million most recent visited sites
		Pool:     &vmPool,
	}, nil
}

func DownloadPAC(ctx context.Context, pacURL string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pacURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get pac %s: %w", pacURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download pac from %s: %w", pacURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected return status code from PAC: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read answer from PAC: %w", err)
	}

	return string(data), nil
}

func (p *Pac) HttpHandleProxy(rawUrl string) (*url.URL, error) {
	host := strings.Split(rawUrl, ":")[1]

	// Cache logic
	entry := p.getFromCache(rawUrl, host)

	proxyFields := strings.Fields(entry)
	switch strings.ToUpper(proxyFields[0]) {
	case "PROXY":
		return url.Parse("http://" + proxyFields[1])
	case "SOCKS", "SOCKS5":
		return url.Parse("socks5://" + proxyFields[1])
	case "DIRECT":
		return nil, nil // no proxy
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyFields[0])
	}
}
