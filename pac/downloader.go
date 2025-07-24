package pac

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func getBytesFromURL(proxyAddr, hostsURL string) ([]byte, error) {
	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy: %w", err)
	}

	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		IdleConnTimeout: time.Second * 30,
	}

	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Get(hostsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download data from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	return scanner.Bytes(), nil
}
