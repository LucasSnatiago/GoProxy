package proxyhandler

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/LucasSnatiago/GoProxy/pac"
)

func handleHTTPS(w http.ResponseWriter, req *http.Request, pacparser *pac.Pac) {
	proxyURL, err := pac.HandleProxy(fmt.Sprintf("https:%s", req.URL), pacparser)
	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	target := req.Host
	if proxyURL == nil {
		DoHTTPSDirectConnection(w, req, target)
		return
	}

	if err := DoHTTPSProxyTunnel(w, req, proxyURL.Host, target); err != nil {
		log.Println("Failed to connect to proxy for HTTPS:", err)
		log.Println("Trying direct connection instead. If it works, means the proxy is not configured correctly...")
		DoHTTPSDirectConnection(w, req, target)
		return
	}
}

func DoHTTPSProxyTunnel(w http.ResponseWriter, r *http.Request, proxyURL string, target string) error {
	server, err := net.DialTimeout("tcp", proxyURL, time.Second*300)
	if err != nil {
		log.Println("Fail to connect to proxy for HTTPS:", err)
		return fmt.Errorf("failed to connect to proxy: %w", err)
	}
	defer server.Close()

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if _, err := server.Write([]byte(connectReq)); err != nil {
		log.Println("Failed to write CONNECT to proxy:", err)
		server.Close()
		return fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return fmt.Errorf("hijacking not supported")
	}

	client, bufrw, err := hj.Hijack()
	if err != nil {
		log.Println("Failed to hijack connection:", err)
		return fmt.Errorf("failed to hijack connection: %w", err)
	}
	defer client.Close()

	exchangeData(client, server, bufrw)
	return nil
}

func DoHTTPSDirectConnection(w http.ResponseWriter, r *http.Request, target string) {
	server, err := net.DialTimeout("tcp", target, time.Second*300)
	if err != nil {
		log.Println("DIRECT failed:", err)
		return
	}
	log.Printf("DIRECT accessed %s\n", target)
	defer server.Close()

	w.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	client, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	exchangeData(client, server, bufrw)
}

func exchangeData(client, server net.Conn, bufrw *bufio.ReadWriter) {
	// Write the buffered data to the server
	go io.Copy(server, bufrw.Reader)

	// Keep the HTTP+TLS connection alive by copying data between client and server
	// until one of them closes the connection.
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(server, client)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(client, server)
		done <- struct{}{}
	}()
	<-done
}
