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
			proxyURL, err := pacparser.HttpHandleProxy(fmt.Sprintf("http://%s", r.Host))
			if err != nil {
				log.Printf("PAC resolution error for %s: %v", r.Host, err)
			} else if proxyURL != nil {
				log.Printf("%s accessed through proxy: %s", r.Host, proxyURL.Host)
			} else {
				log.Printf("%s accessed directly (DIRECT)", r.Host)
			}
			return proxyURL, err
		},
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
	target := req.Host
	proxyURL, err := pacparser.HttpHandleProxy(fmt.Sprintf("https:%s", req.URL))

	if err != nil {
		log.Println("Failed to resolve proxy (HTTPS):", err)
		return
	}

	var server net.Conn
	if proxyURL != nil {
		server, err = net.DialTimeout("tcp", proxyURL.Host, time.Second*30)
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
			log.Printf("Proxy refused CONNECT: %s. Trying DIRECT!", status)

			// --- Experimental support for wrong proxy configs
			server.Close()
			server, err = net.DialTimeout("tcp", target, time.Second*30)
			if err != nil {
				log.Println("DIRECT failed as well:", err)
				return
			}
			log.Printf("%s was securely accessed directly (DIRECT)\n", target)

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

			return

			// --- Better to find a way thats easier to write this recover part
		}

		for {
			line, err := br.ReadString('\n')
			if err != nil || line == "\r\n" {
				break
			}
		}
		log.Printf("%s was securely accessed through proxy: %s\n", target, proxyURL)
	} else {
		server, err = net.DialTimeout("tcp", target, time.Second*30)
		if err != nil {
			log.Println("Fail to connect directly for HTTPS:", err)
			return
		}
		defer server.Close()
		log.Printf("%s was securely accessed directly (DIRECT)\n", target)
	}

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
