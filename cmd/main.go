package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/acentior/go-httpproxy/pkg/httpproxy"
)

func OnError(ctx *httpproxy.Context, where string,
	err *httpproxy.Error, opErr error,
) {
	// Log errors.
	log.Printf("ERR: %s: %s [%s]", where, err, opErr)
}

func OnAccept(ctx *httpproxy.Context, w http.ResponseWriter,
	r *http.Request,
) bool {
	// Handle local request has path "/info"
	if r.Method == "GET" && !r.URL.IsAbs() && r.URL.Path == "/info" {
		w.Write([]byte("This is go-httpproxy."))
		return true
	}
	return false
}

func OnAuth(ctx *httpproxy.Context, authType string, user string, pass string) bool {
	// Auth test user.
	if user == "test" && pass == "1234" {
		return true
	}
	return false
}

func OnConnect(ctx *httpproxy.Context, host string) (
	ConnectAction httpproxy.ConnectAction, newHost string,
) {
	// Apply "Man in the Middle" to all ssl connections. Never change host.
	return httpproxy.ConnectMitm, host
}

func OnRequest(ctx *httpproxy.Context, req *http.Request) (
	resp *http.Response,
) {
	// Log proxying requests.
	log.Printf("INFO: Proxy: %s %s", req.Method, req.URL.String())
	return
}

func OnResponse(ctx *httpproxy.Context, req *http.Request,
	resp *http.Response,
) {
	// Add header "Via: go-httpproxy".
	resp.Header.Add("Via", "go-httpproxy")
}

func main() {
	const PORT = 8080
	// Create a new proxy with default certificate pair.
	prx, err := httpproxy.NewProxy()
	if err != nil {
		log.Printf("Failed to init proxy server. error: %v", err)
	}

	// Set handlers.
	prx.OnError = OnError
	prx.OnAccept = OnAccept
	prx.OnAuth = OnAuth
	prx.OnConnect = OnConnect
	prx.OnRequest = OnRequest
	prx.OnResponse = OnResponse

	// Listen...
	// go http.ListenAndServe(":8080", prx)

	ch := make(chan error, 2)
	go func() {
		log.Printf("Start proxy server on port: %d", PORT)
		err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), prx)
		ch <- err
	}()

	go func() {
		log.Printf("Start proxy server on port: %d", 8081)
		// err := serveHTTPS("./pkg/httpproxy/ca_cert.pem", "./pkg/httpproxy/ca_key.pem")
		err := httpproxy.ServeHTTP(8081)
		ch <- err
	}()

	err = <-ch
	if err != nil {
		log.Printf("Failed to start proxy server. error: %v", err)
	}

	err = <-ch
	if err != nil {
		log.Printf("Failed to start proxy server. error: %v", err)
	}
}
