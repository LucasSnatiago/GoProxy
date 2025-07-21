package pac

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/jackwakefield/gopac"
)

func (p *Pac) HttpHandleProxy(rawUrl string) (*url.URL, error) {
	host := strings.Split(rawUrl, ":")[1]
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
