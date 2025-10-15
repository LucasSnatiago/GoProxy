package proxyhandler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LucasSnatiago/GoProxy/adblock"
	"github.com/LucasSnatiago/GoProxy/pac"
)

func handleLocalSettings(w http.ResponseWriter, r *http.Request, pacparser *pac.Pac, adblock *adblock.AdBlocker) {
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch path[len(path)-1] {
	case "settings":
		fmt.Fprintf(w, "GoProxy is running. Current PAC: %v\n", pacparser)
	case "reload":
		err := pacparser.Reload()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reload PAC: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "PAC reloaded successfully.")
	case "cache":
		cache_entries, err := pacparser.PacCacheToString()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get PAC cache: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Cache hits: %v\tCache misses: %v\n\nCached entries:\n%s", pac.CacheHits(), pac.CacheMisses(), cache_entries)
	case "adblock":
		if adblock != nil {
			fmt.Fprintf(w, "AdBlock is enabled:\n%s", adblock.ToString())
		} else {
			fmt.Fprintln(w, "AdBlock is disabled.")
		}
	case "help":
		fmt.Fprintln(w, "Available commands:\n/settings - Show current settings\n/reload - Reload the PAC script\n/cache - Show PAC cache statistics\n/adblock - Show AdBlock status and entries\n/help - Show this help message")
	default:
		http.Error(w, "Unknown local command", http.StatusNotFound)
	}
}
