package pac

import (
	"fmt"
	"net/url"
	"strings"
)

func HandleProxy(target string, p *Pac) (*url.URL, error) {
	rawUrl := GetFromCache(target, p)

	proxyFields := strings.Fields(rawUrl)
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
