package main

import "C"

import (
	"github.com/acentior/go-httpproxy/pkg/logging"
	goproxy "github.com/acentior/go-httpproxy/pkg/proxy"
	"github.com/acentior/go-httpproxy/pkg/proxy/util"
)

//export RunProxy
func RunProxy(port int, user *C.char, pass *C.char) {
	logger := logging.DefaultLogger()

	proxy := goproxy.NewProxyHttpServer()
	proxyConfig := util.ProxyConfig{
		Port:       uint(port),
		Username:   C.GoString(user),
		Password:   C.GoString(pass),
		CACertPath: "",
		CAKeyPath:  "",
	}
	srvProxy, httpsListener := util.HttpServer(proxy, &proxyConfig)
	if srvProxy == nil || httpsListener == nil {
		logger.Errorw("Faild to configure proxy server", "config", proxyConfig)
		return
	} else {
		logger.Infof("Start to proxy server :%d", proxyConfig.Port)
		srvProxy.Serve(httpsListener)
	}
}

func main() {
}
