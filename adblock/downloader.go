package adblock

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LucasSnatiago/GoProxy/pac"
)

func DownloadStevensBlackBlackList(pacparser *pac.Pac) *AdBlocker {
	data, err := GetBytesFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts", pacparser)
	if err != nil || len(data) == 0 {
		log.Printf("Failed to download Stevens Black List: %v\nTurning adblock off", err)
	}

	entries, err := ParseHostList(bufio.NewScanner(strings.NewReader(string(data))))
	if err != nil || len(entries) == 0 {
		log.Printf("Failed to parse Stevens Black List: %v\nTurning adblock off", err)
	}

	return &AdBlocker{
		Entries: entries,
	}
}

func GetBytesFromURL(link string, p *pac.Pac) ([]byte, error) {
	rawProxyURL := pac.GetFromCache(link, p)
	proxyUrl := strings.Split(rawProxyURL, " ")
	proxyTarget, err := url.Parse(fmt.Sprintf("http://%s", proxyUrl[1]))
	if err != nil {
		log.Printf("Failed to parse proxy URL %s: %v", rawProxyURL, err)
		return nil, err
	}

	proxy := http.ProxyURL(proxyTarget)
	transport := &http.Transport{
		Proxy: proxy,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   300 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", link, err)
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to download %s: %v", link, err)
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
