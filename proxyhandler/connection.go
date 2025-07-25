package proxyhandler

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func DoProxyRequest(proxyURL string, target string) (net.Conn, error) {
	server, err := net.DialTimeout("tcp", proxyURL, time.Second*300)
	if err != nil {
		log.Println("Fail to connect to proxy for HTTPS:", err)
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if _, err := server.Write([]byte(connectReq)); err != nil {
		log.Println("Failed to write CONNECT to proxy:", err)
		server.Close()
		return nil, fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	return server, nil
}

func DoDirectRequest(client net.Conn, target string) {
	server, err := net.DialTimeout("tcp", target, time.Second*300)
	if err != nil {
		log.Println("DIRECT failed:", err)
		return
	}
	log.Printf("DIRECT accessed %s\n", target)
	defer server.Close()

	fmt.Fprintf(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
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
