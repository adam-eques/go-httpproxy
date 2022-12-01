package util

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/acentior/go-httpproxy/pkg/logging"
	goproxy "github.com/acentior/go-httpproxy/pkg/proxy"
	"github.com/acentior/go-httpproxy/pkg/proxy/bandwidth"
	"github.com/acentior/go-httpproxy/pkg/proxy/ext/auth"
)

type ProxyConfig struct {
	Port       uint   `mapstructure:"PROXY_PORT"`
	Addr       string `mapstructure:"PROXY_ADDR"`
	Username   string
	Password   string
	CACertPath string `mapstructure:"PROXY_CA_CERT_PATH"`
	CAKeyPath  string `mapstructure:"PROXY_CA_KEY_PATH"`
}

func HttpServer(proxy *goproxy.ProxyHttpServer, cfg *ProxyConfig) (server *http.Server, listener net.Listener) {
	logger := logging.DefaultLogger()
	verbose := flag.Bool("v", true, "should every proxy request be logged to stdout")
	addr := flag.String("addr", fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port), "proxy listen address")
	flag.Parse()

	// Bandwidth counter
	httpListener, httpsConns, err := bandwidth.InterceptListen("tcp", *addr)
	if err != nil {
		logger.Errorw("proxy.util.HttpsServer failed to create httpListener", "err", err)
		return nil, nil
	}

	proxy.Verbose = *verbose

	// Authenticate middleware
	proxy.OnRequest().Do(auth.Basic("auth", authHandler(httpsConns, cfg.Username, cfg.Password)))
	proxy.OnRequest().HandleConnect(auth.BasicConnect("auth", authHandler(httpsConns, cfg.Username, cfg.Password)))

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return req, req.Response
	})

	httpServer := http.Server{Handler: proxy, Addr: *addr}
	return &httpServer, httpListener
}

// authenticate user and initiate the bandwidth counter.
func authHandler(httpsConns *bandwidth.ConnMap, username, password string) func(req *http.Request, user, passwd string) bool {
	logger := logging.DefaultLogger()
	return func(req *http.Request, user, passwd string) (authorized bool) {
		// Set Username to interceptCon
		authorized = false
		// authenticate
		authorized = authenticate(user, passwd, username, password)
		// initiate the bandWidthCounter
		remoteAddr := req.RemoteAddr
		conn, ok := httpsConns.Find(remoteAddr)
		if ok {
			interceptConn, ok := conn.(*bandwidth.InterceptConn)
			if ok {
				interceptConn.OnClose = func(bytesRead, bytesWritten int) {
					httpsConns.Pop(remoteAddr)
					if authorized {
						bandwidthCount(user, bytesRead, bytesWritten, remoteAddr)
					}
				}
				logger.Infof("\tOnClose handler was set for %s", remoteAddr)
			}
		}
		return
	}
}

// Authenticate with username and password
func authenticate(user, passwd, username, password string) bool {
	return user == username && passwd == password
}

// Handle the counted bandwidth
func bandwidthCount(username string, bytesRead, bytesWritten int, remoteAddr string) {
	logger := logging.DefaultLogger()
	logger.Infof("*** %s read %d, written %d via %s", username, bytesRead, bytesWritten, remoteAddr)
}
