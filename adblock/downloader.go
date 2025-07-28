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
	data, err := GetBytesFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/fakenews-gambling-porn-social/hosts", pacparser)
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
	// Trying directly first
	getReq, err := http.Get(link)
	if err == nil {
		defer getReq.Body.Close()
		return io.ReadAll(getReq.Body)
	}

	// Retrying through proxy if direct request fails
	log.Printf("Failed to get adblock directly. Trying through proxy")

	rawProxyURL := pac.GetFromCache(link, p)
	proxyUrl := strings.Split(rawProxyURL, " ")
	proxyTarget, err := url.Parse(fmt.Sprintf("http://%s", proxyUrl[1]))
	if err != nil {
		log.Printf("Failed to parse proxy URL %s: %v", rawProxyURL, err)
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyTarget)},
		Timeout:   300 * time.Second,
	}

	resp, err := client.Get(link)
	if err != nil {
		log.Printf("Failed to download %s: %v", link, err)
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
