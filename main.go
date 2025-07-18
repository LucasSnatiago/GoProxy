package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/armon/go-socks5"
	"github.com/jackwakefield/gopac"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	listenAddr := flag.String("l", "localhost", "ip to listen on")
	httpPort := flag.Int("p", 3128, "HTTP/HTTPS port to listen on")
	socksPort := flag.Int("s", 8010, "SOCKS5 port to listen on")
	pacUrl := flag.String("C", "", "Proxy Auto Configuration URL")
	flag.Parse()

	parser := new(gopac.Parser)

	if pacUrl == nil || *pacUrl == "" {
		fmt.Println("Please specify pac url using -C")
		os.Exit(1)
	}

	if err := parser.ParseUrl(*pacUrl); err != nil {
		log.Fatalf("Failed to parse PAC (%s)", err)
	}

	httpAddr := net.JoinHostPort(*listenAddr, fmt.Sprint(*httpPort))
	ln, err := net.Listen("tcp", httpAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Proxy HTTP listening on ", httpAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("Failed to accept new connection:", err)
				continue
			}

			go handleHTTP(conn, parser)
		}
	}()

	// Socks5
	socks5addr := net.JoinHostPort(*listenAddr, fmt.Sprint(*socksPort))
	log.Println("Proxy SOCKS5 listening on ", socks5addr)
	go func() {
		conf := &socks5.Config{
			Dial:   httpConnectDialer(httpAddr),
			Logger: log.New(os.Stdout, "[SOCKS5] ", log.LstdFlags),
		}
		server, err := socks5.New(conf)
		if err != nil {
			panic(err)
		}

		// Create SOCKS5 proxy on localhost port 8000
		if err := server.ListenAndServe("tcp", socks5addr); err != nil {
			log.Println("Failed to start socks5 server:", err)
		}
	}()

	// Use CTRL + C to stop process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.Println("Program running. Press Ctrl+C to stop.")
	<-sigChan
	fmt.Println("Received Ctrl+C. Turning off...")
}
