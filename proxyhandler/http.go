package proxyhandler

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LucasSnatiago/GoProxy/adblock"
	"github.com/LucasSnatiago/GoProxy/pac"
)

type ProxyHandler struct {
	PacParser *pac.Pac
	Adblocker *adblock.AdBlocker
}

func HandleHTTPConnection(conn net.Conn, pacparser *pac.Pac, adblock *adblock.AdBlocker) {
	defer conn.Close()

	proxyHandler := &ProxyHandler{
		PacParser: pacparser,
		Adblocker: adblock,
	}

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		if err != io.EOF {
			log.Println("Fail to read request:", err)
		}
		return
	}

	if proxyHandler.Adblocker != nil {
		// Drop connection if the host appears on the adblock list
		host := strings.Split(req.Host, ":")
		if proxyHandler.Adblocker.CheckIfAppearsOnAdblockList(host[0]) {
			log.Printf("Blocked request to %s due to adblock rules", req.Host)
			writeHTTPError(conn, http.StatusForbidden, "Forbidden")
			return
		}
	}

	if req.Method == http.MethodConnect {
		handleHTTPS(conn, req, proxyHandler)
	} else {
		handlePlainHTTP(conn, req, proxyHandler)
	}
}

func handlePlainHTTP(client net.Conn, req *http.Request, proxyHandler *ProxyHandler) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	trnprt := &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return pac.HandleProxy(fmt.Sprintf("http://%s", r.Host), proxyHandler.PacParser)
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

func handleHTTPS(client net.Conn, req *http.Request, proxyHandler *ProxyHandler) {
	proxyURL, err := pac.HandleProxy(fmt.Sprintf("https:%s", req.URL), proxyHandler.PacParser)
	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	target := req.Host
	if proxyURL == nil {
		DoHTTPSDirectConnection(client, target)
		return
	}

	if err := DoHTTPSProxyTunnel(client, proxyURL.Host, target); err != nil {
		log.Println("Failed to connect to proxy for HTTPS:", err)
		log.Println("Trying direct connection instead. If it works, means the proxy is not configured correctly...")
		DoHTTPSDirectConnection(client, target)
		return
	}
}
