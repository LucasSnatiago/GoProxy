package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LucasSnatiago/GoProxy/pac"
	"github.com/LucasSnatiago/GoProxy/proxyhandler"
	"github.com/armon/go-socks5"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	listenAddr := flag.String("l", "localhost", "ip to listen on")
	httpPort := flag.Int("p", 3128, "HTTP/HTTPS port to listen on")
	socksPort := flag.Int("s", 8010, "SOCKS5 port to listen on")
	pacUrl := flag.String("C", "", "Proxy Auto Configuration URL")
	ttlSeconds := flag.Int64("S", int64(time.Minute)*5, "sets how long (in seconds) for the cache to keep the entries")
	logMessages := flag.Bool("v", false, "if you set this flag it will enable console output for every request")
	flag.Parse()

	// Disabling log messages
	if !*logMessages {
		log.SetOutput(io.Discard)
	}

	// Pac file is mandatory
	if pacUrl == nil || *pacUrl == "" {
		fmt.Println("Please specify pac url using -C")
		os.Exit(1)
	}

	ctx := context.Background()

	// Proxy Auto Config
	pacScript, err := pac.DownloadPAC(ctx, *pacUrl)
	if err != nil {
		fmt.Println("Failed to parse PAC:", err)
		os.Exit(2)
	}

	pacparser, err := pac.NewPac(pacScript, time.Duration(*ttlSeconds))
	if err != nil {
		fmt.Println("Failed to create pac parser:", err)
		os.Exit(3)
	}

	httpAddr := net.JoinHostPort(*listenAddr, fmt.Sprint(*httpPort))
	ln, err := net.Listen("tcp", httpAddr)
	if err != nil {
		fmt.Println("Failed to start http proxy:", err)
		os.Exit(4)
	}

	fmt.Println("Proxy HTTP listening on", httpAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("Failed to accept new connection:", err)
				continue
			}

			go proxyhandler.HandleHTTP(conn, pacparser)
		}
	}()

	// Socks5
	socks5addr := net.JoinHostPort(*listenAddr, fmt.Sprint(*socksPort))
	go func() {
		conf := &socks5.Config{
			Dial:   proxyhandler.HttpConnectDialer(httpAddr, time.Second*30),
			Logger: log.New(os.Stdout, "[SOCKS5] ", log.LstdFlags),
		}
		server, err := socks5.New(conf)
		if err != nil {
			fmt.Println("Failed to create socks5 object:", err)
		}

		fmt.Println("Proxy SOCKS5 listening on", socks5addr)
		err = server.ListenAndServe("tcp", socks5addr)
		if err != nil {
			fmt.Println("failed to start socks5 server:", err)
		}
	}()

	// Use CTRL + C to stop process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Program running. Press Ctrl+C to stop.")
	<-sigChan
	fmt.Println("Received Ctrl+C. Turning off...")
}
