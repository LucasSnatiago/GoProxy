package pac

import (
	"log"

	"github.com/jackwakefield/gopac"
)

// It tries to retrieve the URL from the cache if it fails it calls an OttoVM
// and retrieves the url directly from the proxy
func (p *Pac) getFromCache(rawUrl, host string) string {
	entry, ok := p.PacCache.Get(host)

	if !ok {
		vm := p.Get().(*gopac.Parser)
		defer p.Put(vm)

		pacrequest, err := vm.FindProxy(rawUrl, host)
		if err != nil {
			log.Printf("Failed to find proxy entry (%s)", err)
		}
		entry = pacrequest

		p.PacCache.Add(host, entry)
	}

	return entry
}
