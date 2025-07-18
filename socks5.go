package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

// Get network package from socks and transform it in a http proxy package
func httpConnectDialer(proxyHTTPAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := net.Dial(network, proxyHTTPAddr)
		if err != nil {
			return nil, err
		}

		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", addr, addr)
		if _, err := conn.Write([]byte(connectReq)); err != nil {
			conn.Close()
			return nil, err
		}

		br := bufio.NewReader(conn)
		status, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, err
		}
		if !strings.HasPrefix(status, "HTTP/1.1 200") {
			conn.Close()
			return nil, errors.New("proxy HTTP rejected CONNECT: " + strings.TrimSpace(status))
		}

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				conn.Close()
				return nil, err
			}
			if line == "\r\n" {
				break
			}
		}

		log.Println("SOCKS5 proxy: ", addr)
		return conn, nil
	}
}
