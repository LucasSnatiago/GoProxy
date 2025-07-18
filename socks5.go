package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Get network package from socks and transform it in a http proxy package
func httpConnectDialer(proxyHTTPAddr string, dialTimeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: dialTimeout,
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := d.DialContext(ctx, network, proxyHTTPAddr)
		if err != nil {
			return nil, err
		}

		if dl, ok := ctx.Deadline(); ok {
			conn.SetDeadline(dl)
		}

		fmt.Fprintf(conn,
			"CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n",
			addr, addr,
		)

		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			conn.Close()
			return nil, fmt.Errorf("proxy rejected CONNECT: %s", resp.Status)
		}

		conn.SetDeadline(time.Time{})
		log.Printf("HTTP CONNECT tunnel established to %s via %s", addr, proxyHTTPAddr)
		return conn, nil
	}
}
