package proxyhandler

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LucasSnatiago/GoProxy/pac"
)

func HandleHTTPConnection(conn net.Conn, pacparser *pac.Pac) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Fail to read request:", err)
		return
	}

	if req.Method == "CONNECT" {
		handleHTTPS(conn, req, pacparser)
	} else {
		handlePlainHTTP(conn, req, pacparser)
	}
}

func handlePlainHTTP(client net.Conn, req *http.Request, pacparser *pac.Pac) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	trnprt := &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return pac.HandleProxy(fmt.Sprintf("http://%s", r.Host), pacparser)
		},
	}

	clientHTTP := &http.Client{
		Timeout:   300 * time.Second,
		Transport: trnprt,
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

func handleHTTPS(client net.Conn, req *http.Request, pacparser *pac.Pac) {
	proxyURL, err := pac.HandleProxy(fmt.Sprintf("https:%s", req.URL), pacparser)
	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	target := req.Host
	if proxyURL == nil {
		DoDirectRequest(client, target)
		return
	}

	server, err := DoProxyRequest(proxyURL.Host, target)
	if err != nil {
		log.Println("Failed to connect to proxy for HTTPS:", err)
		return
	}
	defer server.Close()

	_ = readHTTPData(client, server, target)
}

func readHTTPData(client, server net.Conn, target string) []byte {
	br := bufio.NewReader(server)
	status, err := br.ReadString('\n')
	if err != nil || !strings.Contains(status, "200") {
		log.Printf("Proxy refused CONNECT: %s. Trying DIRECT!", status)

		// --- Experimental support for wrong configured proxies
		DoDirectRequest(client, target)
		return nil
	}

	var resp string
	for {
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		resp += line
	}

	return []byte(resp)
}
