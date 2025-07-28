package pac

import (
	"log"
	"net"
	"net/url"

	"github.com/LucasSnatiago/gopac"
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

	entry, ok := pac.PacCache.Get(target)
	if !ok {
		vm := pac.Get().(*gopac.Parser)
		defer pac.Put(vm)

		pacrequest, err := vm.FindProxy(rawUrl, target)
		if err != nil {
			log.Printf("Failed to find proxy entry (%s)", err)
		}
		entry = pacrequest

		pac.PacCache.Add(target, entry)
	}

	return entry
}
