package main

import (
	"github.com/acentior/go-httpproxy/pkg/logging"
	goproxy "github.com/acentior/go-httpproxy/pkg/proxy"
	"github.com/acentior/go-httpproxy/pkg/proxy/util"
)

func main() {
	logger := logging.DefaultLogger()

	proxy := goproxy.NewProxyHttpServer()
	proxyConfig := util.ProxyConfig{
		Port:       8080,
		Addr:       "127.0.0.1",
		Username:   "admin",
		Password:   "123456",
		CACertPath: "",
		CAKeyPath:  "",
	}
	srvProxy, httpsListener := util.HttpServer(proxy, &proxyConfig)
	if srvProxy == nil || httpsListener == nil {
		logger.Errorw("Faild to configure proxy server", "config", proxyConfig)
		return
	} else {
		logger.Infof("Start to proxy server %s:%d", proxyConfig.Addr, proxyConfig.Port)
		srvProxy.Serve(httpsListener)
	}
}
