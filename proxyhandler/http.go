package proxyhandler

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/LucasSnatiago/GoProxy/adblock"
	"github.com/LucasSnatiago/GoProxy/pac"
)

func HandleHTTPConnection(w http.ResponseWriter, r *http.Request, pacparser *pac.Pac, adblock *adblock.AdBlocker) {
	if shouldBlockAds(r, adblock) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
		return
	}

	// Add the proxy authentication if provided
	if pacparser.Auth != nil {
		r.SetBasicAuth(pacparser.Auth.User, pacparser.Auth.Password)
	}

	if r.Method == http.MethodConnect {
		handleHTTPS(w, r, pacparser)
	} else {
		handlePlainHTTP(w, r, pacparser)
	}
}

func handlePlainHTTP(w http.ResponseWriter, req *http.Request, pacparser *pac.Pac) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	trnprt := &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return pac.HandleProxy(fmt.Sprintf("http://%s", r.Host), pacparser)
		},
	}

	clientHTTP := &http.Client{
		Timeout:   300 * time.Second,
		Transport: trnprt,
	}

	resp, err := clientHTTP.Do(req)
	if err != nil {
		log.Printf("Failed to send request to %s: %v", req.URL, err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to read response body for %s: %v", req.URL, err)))
		return
	}
	if _, err := w.Write(body); err != nil {
		log.Printf("Failed to write response for %s: %v", req.URL, err)
	}
}

func shouldBlockAds(req *http.Request, adblocker *adblock.AdBlocker) bool {
	if adblocker != nil {
		// Drop connection if the host appears on the adblock list
		host, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			host = req.Host // If no port is specified, use the whole host
		}
		if adblocker.CheckIfAppearsOnAdblockList(host) {
			log.Printf("Blocked request to %s due to adblock rules", req.Host)
			return true
		}
	}
	return false
}
