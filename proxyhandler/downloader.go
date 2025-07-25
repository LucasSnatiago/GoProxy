package proxyhandler

import (
	"log"

	"github.com/LucasSnatiago/GoProxy/pac"
)

func GetBytesFromURL(link string, p *pac.Pac) ([]byte, error) {
	conn, err := DoProxyRequest(pac.GetFromCache(link, p), link)
	if err != nil {
		log.Printf("Failed to connect to proxy for %s: %v", link, err)
		return nil, err
	}
	defer conn.Close()

	data := readHTTPData(conn, conn, link)
	return data, nil
}
