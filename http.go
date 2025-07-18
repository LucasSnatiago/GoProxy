package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackwakefield/gopac"
)

var (
	vmLock sync.Mutex
)

func handleHTTP(conn net.Conn, pacparser *gopac.Parser) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Fail to read request:", err)
		return
	}

	if req.Method == "CONNECT" {
		handleHTTPS(conn, req.Host, pacparser)
	} else {
		handlePlainHTTP(conn, req, pacparser)
	}
}

func HttpHandleProxy(host string, pacparser *gopac.Parser) (*url.URL, error) {
	entry, ok := pacCache.Get(host)

	if !ok {
		vmLock.Lock()
		defer vmLock.Unlock()
		pacrequest, err := pacparser.FindProxy("", host)
		if err != nil {
			log.Printf("Failed to find proxy entry (%s)", err)
		}

		entry = pacrequest
		pacCache.Add(host, pacrequest)
	}

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

func handlePlainHTTP(client net.Conn, req *http.Request, pacparser *gopac.Parser) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	proxyURL, err := HttpHandleProxy(req.Host, pacparser)
	if err != nil {
		log.Printf("Failed to resolve proxy for %s: %v", req.URL, err)
		writeHTTPError(client, http.StatusBadGateway, "Bad Gateway")
		return
	}

	if proxyURL != nil {
		log.Printf("%s was accessed through proxy: %s", req.URL, proxyURL.Host)
	} else {
		log.Printf("%s was directly accessed (DIRECT)", req.URL)
	}

	clientHTTP := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				proxyURL, err := HttpHandleProxy(r.Host, pacparser)
				if err != nil {
					log.Printf("PAC resolution error for %s: %v", r.URL.Host, err)
				} else if proxyURL != nil {
					log.Printf("%s accessed through proxy: %s", r.URL.String(), proxyURL.Host)
				} else {
					log.Printf("%s accessed directly (DIRECT)", r.URL.String())
				}
				return proxyURL, err
			},
			DisableKeepAlives: true,
		},
	}

	resp, err := clientHTTP.Do(req)
	if err != nil {
		log.Printf("Failed to send request to %s: %v", req.URL, err)
		writeHTTPError(client, http.StatusBadGateway, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	if err := resp.Write(client); err != nil {
		log.Printf("Failed to write response for %s: %v", req.URL, err)
	}
}

func writeHTTPError(conn net.Conn, statusCode int, statusText string) {
	body := fmt.Sprintf("%d %s", statusCode, statusText)
	fmt.Fprintf(conn,
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		statusCode, statusText, len(body), body)
}

func handleHTTPS(client net.Conn, target string, pacparser *gopac.Parser) {
	rawUrlWithPort, err := url.Parse(target)
	if err != nil {
		log.Println("failed to parse HTTPS url", err)
	}
	rawUrl := strings.Split(rawUrlWithPort.String(), ":")[0]

	proxyURL, err := HttpHandleProxy(rawUrl, pacparser)

	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	var server net.Conn

	if proxyURL != nil {
		server, err = net.Dial("tcp", proxyURL.Host)
		if err != nil {
			log.Println("Fail to connect to proxy for HTTPS:", err)
			return
		}

		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
		if _, err := server.Write([]byte(connectReq)); err != nil {
			log.Println("Failed to write CONNECT to proxy:", err)
			server.Close()
			return
		}

		br := bufio.NewReader(server)
		status, err := br.ReadString('\n')
		if err != nil || !strings.Contains(status, "200") {
			log.Println("Proxy refused CONNECT:", status)
			server.Close()
			return
		}

		for {
			line, err := br.ReadString('\n')
			if err != nil || line == "\r\n" {
				break
			}
		}
		log.Printf("%s was securely accessed through proxy: %s\n", target, proxyURL)
	} else {
		server, err = net.Dial("tcp", target)
		if err != nil {
			log.Println("Fail to connect directly for HTTPS:", err)
			return
		}
		log.Printf("%s was securely accessed directly (DIRECT)\n", target)
	}

	defer server.Close()

	fmt.Fprintf(client, "HTTP/1.1 200 Connection Established\r\n\r\n")

	go io.Copy(server, client)
	io.Copy(client, server)
}
