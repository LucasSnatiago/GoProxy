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

func HandleHTTP(conn net.Conn, pacparser *pac.Pac) {
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
			return pacparser.HttpHandleProxy(fmt.Sprintf("http://%s", req.Host))
		},
		//Proxy: func(r *http.Request) (*url.URL, error) {
		//	proxyURL, err := pacparser.HttpHandleProxy(fmt.Sprintf("http://%s", r.Host))
		//	if err != nil {
		//		log.Printf("PAC resolution error for %s: %v", r.Host, err)
		//	} else if proxyURL != nil {
		//		log.Printf("%s accessed through proxy: %s", r.Host, proxyURL.Host)
		//	} else {
		//		log.Printf("%s accessed directly (DIRECT)", r.Host)
		//	}
		//	return proxyURL, err
		//},
	}

	clientHTTP := &http.Client{
		Timeout:   30 * time.Second,
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
	proxyURL, err := pacparser.HttpHandleProxy(fmt.Sprintf("https:%s", req.URL))
	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	target := req.Host
	if proxyURL == nil {
		doDirectRequest(client, target)
		return
	}

	server, err := doProxyRequest(proxyURL, target)
	if err != nil {
		log.Println("Failed to connect to proxy for HTTPS:", err)
		return
	}
	defer server.Close()

	br := bufio.NewReader(server)
	status, err := br.ReadString('\n')
	if err != nil || !strings.Contains(status, "200") {
		log.Printf("Proxy refused CONNECT: %s. Trying DIRECT!", status)

		// --- Experimental support for wrong configured proxies
		doDirectRequest(client, target)
		return
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}
}
