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

	log.Printf("INFO: Proxy: %s %s %s", req.URL.Scheme, req.Method, req.URL.String())
	return
}

func OnResponse(ctx *httpproxy.Context, req *http.Request,
	resp *http.Response,
) {
	// Add header "Via: go-httpproxy".
	resp.Header.Add("Via", "go-httpproxy")
}

func SizeCount(user string, read, written int) {
	fmt.Printf("%s read %v bytes, wrote %v bytes\n", user, read, written)
}

func main() {
	const PORT = 80

	// cert, err := os.ReadFile("./ca_cert.pem")
	// if err != nil {
	// 	fmt.Println("Failed to load cert")
	// }
	// key, err := os.ReadFile("./ca_key.pem")
	// if err != nil {
	// 	fmt.Println("Failed to load key")
	// }

	var prx *httpproxy.Proxy
	// // Create a new proxy with default certificate pair.
	// if len(cert) == 0 || len(key) == 0 {
	// 	fmt.Println("ssl with default cert & key")
	// 	prx, err = httpproxy.NewProxy()
	// } else {
	// 	fmt.Println("ssl with custom cert & key")
	// 	prx, err = httpproxy.NewProxyCert(cert, key)
	// }
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

	prx.SizeCount = SizeCount

	// Listen...
	// go http.ListenAndServe(":8080", prx)

	ch := make(chan error, 2)
	// go func() {
	// 	log.Printf("Start proxy server on port: %d", PORT)
	// 	err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), prx)
	// 	ch <- err
	// }()

	addr := fmt.Sprintf(":%d", PORT)

	go func() {
		httpsListener, httpsConns, err := httpproxy.InterceptListen("tcp", addr)
		if err != nil || prx == nil {
			return
		}

		prx.HttpsConns = httpsConns

		server := &http.Server{Handler: prx, Addr: addr}

		log.Printf("Start proxy server on port: %d", PORT)
		err = server.Serve(httpsListener)

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
