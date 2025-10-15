package pac

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/LucasSnatiago/gopac"
)

var (
	cacheHits   uint64
	cacheMisses uint64
)

// It tries to retrieve the URL from the cache if it fails it calls an OttoVM
// and retrieves the url directly from the proxy
func GetFromCache(rawUrl string, pac *Pac) string {
	url, err := url.Parse(rawUrl)
	if err != nil {
		log.Println("failed to parse url: ", rawUrl)
	}

	// Remove port if it exists
	target, _, err := net.SplitHostPort(url.Host)
	if err != nil {
		target = url.Host // If no port is specified, use the whole host
	}

	// Do not cache empty targets
	if strings.TrimSpace(target) == "" {
		return "DIRECT"
	}

	// Check if its an IP address and skip the cache
	ip := net.ParseIP(target)
	if ip != nil {
		return fmt.Sprintf("DIRECT %s", target)
	}

	entry, ok := pac.PacCache.Get(target)
	if !ok {
		vm := pac.Get().(*gopac.Parser)
		defer pac.Put(vm)

		pacrequest, err := vm.FindProxy(rawUrl, target)
		if err != nil {
			log.Printf("Failed to find proxy entry (%s)", err)
		}
		entry = pacrequest

		atomic.AddUint64(&cacheMisses, 1)
		log.Printf("%s accessed for: %s\n", entry, target)
		pac.PacCache.Add(target, entry)
	} else {
		atomic.AddUint64(&cacheHits, 1)
	}

	return entry
}

func startCacheStatsLogger() {
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			hits := atomic.LoadUint64(&cacheHits)
			misses := atomic.LoadUint64(&cacheMisses)
			if hits != 0 {
				log.Printf("Cache hits: %v, misses: %v | %v%% of cache hits\n", hits, misses, hits*100/(hits+misses))
			}
		}
	}()
}

func CacheHits() uint64 {
	return atomic.LoadUint64(&cacheHits)
}

func CacheMisses() uint64 {
	return atomic.LoadUint64(&cacheMisses)
}
