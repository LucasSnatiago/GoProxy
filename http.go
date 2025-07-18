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
	"time"

	"github.com/jackwakefield/gopac"
)

func handleHTTP(conn net.Conn, pacparser *gopac.Parser) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Fail to read request:", err)
		return
	}

	if strings.Compare(req.Method, "CONNECT") == 0 {
		handleHTTPS(conn, reader, req.Host)
	} else {
		handlePlainHTTP(conn, req, pacparser)
	}
}

func HttpHandleProxy(req *http.Request, pacparser *gopac.Parser) (*url.URL, error) {
	entry, err := pacparser.FindProxy("", req.Host)
	if err != nil {
		log.Fatalf("Failed to find proxy entry (%s)", err)
	}

	proxyFields := strings.Fields(entry)
	if len(proxyFields) < 2 {
		return nil, fmt.Errorf("invalid proxy entry: %s", entry)
	}

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
	defer client.Close()

	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	proxyURL, err := HttpHandleProxy(req, pacparser)
	if err != nil {
		log.Println("Failed to resolve proxy:", err)
		return
	}

	if proxyURL != nil {
		log.Printf("%s was accessed through proxy: %s\n", req.URL.String(), proxyURL.Host)
	} else {
		log.Printf("%s was directly accessed (DIRECT)\n", req.URL.String())
	}

	transport := &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
	}

	clientHTTP := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	resp, err := clientHTTP.Do(req)
	if err != nil {
		log.Println("Failed to send request through proxy:", err)
		return
	}
	defer resp.Body.Close()

	err = resp.Write(client)
	if err != nil {
		log.Println("Fail to write answer:", err)
		return
	}
}

func handleHTTPS(client net.Conn, reader *bufio.Reader, target string) {
	server, err := net.Dial("tcp", target)
	if err != nil {
		log.Println("Fail to connet to target HTTPS:", err)
		return
	}

	fmt.Fprintf(client, "HTTP/1.1 200 Connection Established\r\n\r\n")

	log.Printf("%s was securely accessed through proxy: %s\n", target, client.RemoteAddr().String())
	go io.Copy(server, reader)
	io.Copy(client, server)
}
