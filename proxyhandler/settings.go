package proxyhandler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LucasSnatiago/GoProxy/pac"
)

func handleLocalSettings(w http.ResponseWriter, r *http.Request, pacparser *pac.Pac) {
	if strings.Contains(r.URL.Path, "settings") {
		fmt.Fprintf(w, "GoProxy is running. Current PAC: %v\n", pacparser)

	} else if strings.Contains(r.URL.Path, "reload") {
		err := pacparser.Reload()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reload PAC: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "PAC reloaded successfully.")

	} else if strings.Contains(r.URL.Path, "cache") {
		fmt.Fprintf(w, "Cache hits: %v\nCache misses: %v\n", pac.CacheHits(), pac.CacheMisses())

	} else {
		http.Error(w, "Unknown local command", http.StatusNotFound)

	}
}
