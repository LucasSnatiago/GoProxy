package pac

import (
	"fmt"
	"io"
	"net/http"
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
