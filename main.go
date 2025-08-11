package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LucasSnatiago/GoProxy/adblock"
	"github.com/LucasSnatiago/GoProxy/pac"
	"github.com/LucasSnatiago/GoProxy/proxyhandler"
	"github.com/elazarl/goproxy"
	"github.com/things-go/go-socks5"
)

func main() {
	log.SetFlags(log.Lshortfile)
	pacUrl := flag.String("C", "http://wpad/wpad.dat", "Proxy Auto Configuration URL")
	listenAddr := flag.String("l", "localhost", "ip to listen on")
	httpPort := flag.Int("p", 3128, "HTTP/HTTPS port to listen on")
	osHttpPort := flag.Int("P", 3129, "HTTP port to listen on for OS (ex.: Windows)")
	socksPort := flag.Int("s", 8010, "SOCKS5 port to listen on")
	username := flag.String("user", "", "username for authentication")
	password := flag.String("pass", "", "password for authentication")
	ttlSeconds := flag.Int64("S", 5*60, "sets how long (in seconds) for the cache to keep the entries - default is 5 minutes")
	logMessages := flag.Bool("v", false, "if you set this flag it will enable console output for every request")
	adblockEnabled := flag.Bool("a", false, "enable adblock usage on the proxy")
	displayVersion := flag.Bool("version", false, "display GoProxy current version")
	flag.Parse()

	// Display version and exit
	if *displayVersion {
		fmt.Println(DisplayVersion())
		os.Exit(0)
	}

	// Disabling log messages
	if !*logMessages {
		log.SetOutput(io.Discard)
	}

	// Pac file is mandatory
	if pacUrl == nil || *pacUrl == "" {
		fmt.Println("Please specify pac url using -C")
		os.Exit(1)
	}

	// Proxy Auto Config
	pacScript, err := pac.DownloadPAC(*pacUrl)
	if err != nil {
		fmt.Println("Failed to parse PAC:", err)
		os.Exit(2)
	}

	pacparser, err := pac.NewPac(pacScript, time.Second*time.Duration(*ttlSeconds))
	if err != nil {
		fmt.Println("Failed to create pac parser:", err)
		os.Exit(3)
	}
	pacparser.SetAuth(*username, *password)

	fmt.Println("Running GoProxy version:", version)

	// Adblock
	var adblocker *adblock.AdBlocker
	if *adblockEnabled {
		adblocker = adblock.NewAdblock(pacparser)
		if adblocker == nil {
			fmt.Println("AdBlock is disabled, something went wrong.")
		} else {
			log.Println("Adblock up and running")
		}
	}

	httpAddr := net.JoinHostPort(*listenAddr, fmt.Sprint(*httpPort))

	// Proxy HTTP for Browsers
	fmt.Println("Proxy HTTP for browsers listening on", httpAddr)
	go func() {
		err = http.ListenAndServe(httpAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxyhandler.HandleHTTPConnection(w, r, pacparser, adblocker)
		}))
	}()
	// If fail to start HTTP server, exit
	if err != nil {
		fmt.Println("Failed to start http proxy:", err)
		os.Exit(4)
	}

	// Proxy HTTPS for OS
	fmt.Println("Proxy HTTPS for your operational system listening on", net.JoinHostPort(*listenAddr, fmt.Sprint(*osHttpPort)))
	go func() {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = true

		if *adblockEnabled {
			for _, filter := range adblocker.Entries {
				proxy.OnRequest(goproxy.UrlIs(filter)).HandleConnect(goproxy.AlwaysReject)
			}
		}

		log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *listenAddr, *osHttpPort), proxy))
	}()

	// Socks5
	go func() {
		socks5addr := net.JoinHostPort(*listenAddr, fmt.Sprint(*socksPort))
		server := socks5.NewServer(
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "[SOCKS5] ", log.Lshortfile))),
			socks5.WithDial(proxyhandler.HttpConnectDialer(httpAddr, time.Second*300)),
		)

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
