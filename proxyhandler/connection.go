package proxyhandler

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func DoHTTPSProxyTunnel(client net.Conn, proxyURL string, target string) error {
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

	exchangeData(client, server)
	return nil
}

func DoHTTPSDirectConnection(client net.Conn, target string) {
	server, err := net.DialTimeout("tcp", target, time.Second*300)
	if err != nil {
		log.Println("DIRECT failed:", err)
		return
	}
	log.Printf("DIRECT accessed %s\n", target)
	defer server.Close()

	fmt.Fprintf(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
	exchangeData(client, server)
}

func exchangeData(client, server net.Conn) {
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
