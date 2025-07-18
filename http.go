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
)

var (
	vmLock sync.Mutex
)

func handleHTTP(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Fail to read request:", err)
		return
	}

	if req.Method == "CONNECT" {
		handleHTTPS(conn, req)
	} else {
		handlePlainHTTP(conn, req)
	}
}

func HttpHandleProxy(rawUrl string) (*url.URL, error) {
	host := strings.Split(rawUrl, ":")[1]
	entry, ok := pacCache.Get(host)

	if !ok {
		vmLock.Lock()
		pacrequest, err := pacparser.FindProxy(rawUrl, host)
		if err != nil {
			log.Printf("Failed to find proxy entry (%s)", err)
		}

		entry = pacrequest
		pacCache.Add(host, pacrequest)
		vmLock.Unlock()
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

func handlePlainHTTP(client net.Conn, req *http.Request) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	clientHTTP := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				proxyURL, err := HttpHandleProxy(fmt.Sprintf("http://%s", r.Host))
				if err != nil {
					log.Printf("PAC resolution error for %s: %v", r.Host, err)
				} else if proxyURL != nil {
					log.Printf("%s accessed through proxy: %s", r.Host, proxyURL.Host)
				} else {
					log.Printf("%s accessed directly (DIRECT)", r.Host)
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

func handleHTTPS(client net.Conn, req *http.Request) {
	target := req.Host
	proxyURL, err := HttpHandleProxy(fmt.Sprintf("https:%s", req.URL))

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
