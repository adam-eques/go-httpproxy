package httpproxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type ConnMap struct {
	conns map[string]net.Conn
	m     sync.Mutex
}

func NewConnMap() *ConnMap {
	return &ConnMap{conns: map[string]net.Conn{}}
}

func (cm *ConnMap) Pop(remoteAddr string) (net.Conn, bool) {
	fmt.Printf("ConnMap popping '%v'\n", remoteAddr)
	cm.m.Lock()
	defer cm.m.Unlock()
	c, ok := cm.conns[remoteAddr]
	delete(cm.conns, remoteAddr)
	return c, ok
}

func (cm *ConnMap) Push(conn net.Conn) {
	fmt.Printf("ConnMap pushing '%v'\n", conn.RemoteAddr().String())
	cm.m.Lock()
	defer cm.m.Unlock()
	cm.conns[conn.RemoteAddr().String()] = conn
}

func (cm *ConnMap) Find(remoteAddr string) (net.Conn, bool) {
	fmt.Printf("ConnMap Finding '%v'\n", remoteAddr)
	cm.m.Lock()
	defer cm.m.Unlock()
	c, ok := cm.conns[remoteAddr]
	// delete(cm.conns, remoteAddr)
	return c, ok
}

type InterceptListener struct {
	realListener net.Listener
	connMap      *ConnMap
}

// InterceptListen creates and returns a net.Listener via net.Listen, which is wrapped with an intercepter, which counts Conn read and write bytes. If you want a `grove.NewCacheHandler` to be able to count in and out bytes per remap rule in the stats interface, it must be served with a listener created via InterceptListen or InterceptListenTLS.
func InterceptListen(network, laddr string) (net.Listener, *ConnMap, error) {
	l, err := net.Listen(network, laddr)
	if err != nil {
		return l, nil, err
	}
	connMap := NewConnMap()
	return &InterceptListener{realListener: l, connMap: connMap}, connMap, nil
}

func InterceptListenTLS(net, laddr, certFile, keyFile string) (net.Listener, *ConnMap, error) {
	interceptListener, connMap, err := InterceptListen(net, laddr)
	if err != nil {
		return nil, nil, err
	}

	config := &tls.Config{NextProtos: []string{"h2"}}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}

	tlsLst := tls.NewListener(interceptListener, config)
	return tlsLst, connMap, nil
}

func (l *InterceptListener) Accept() (net.Conn, error) {
	c, err := l.realListener.Accept()
	if err != nil {
		return c, err
	}
	interceptConn := &InterceptConn{realConn: c}
	l.connMap.Push(interceptConn)
	fmt.Printf("InterceptListener Accept conn %T\n", c)
	return interceptConn, nil
}

func (l *InterceptListener) Close() error {
	return l.realListener.Close()
}

func (l *InterceptListener) Addr() net.Addr {
	return l.realListener.Addr()
}

type InterceptConn struct {
	realConn     net.Conn
	bytesRead    int
	bytesWritten int
	requested    bool
	responsed    bool

	OnClose func(bytesRead, bytesWritten int)
}

func (c *InterceptConn) Requested() {
	c.requested = true
}

func (c *InterceptConn) Responsed() {
	c.responsed = true
}

func (c *InterceptConn) BytesRead() int {
	return c.bytesRead
}

func (c *InterceptConn) BytesWritten() int {
	return c.bytesWritten
}

func (c *InterceptConn) Read(b []byte) (n int, err error) {
	n, err = c.realConn.Read(b)
	c.bytesRead += n
	return
}

func (c *InterceptConn) Write(b []byte) (n int, err error) {
	n, err = c.realConn.Write(b)
	c.bytesWritten += n
	return
}

func (c *InterceptConn) Close() error {
	if !c.requested {
		c.bytesRead = 0
	}
	if !c.responsed {
		c.bytesWritten = 0
	}
	if c.OnClose != nil {
		c.OnClose(c.bytesRead, c.bytesWritten)
	}
	return c.realConn.Close()
}

func (c *InterceptConn) LocalAddr() net.Addr {
	return c.realConn.LocalAddr()
}

func (c *InterceptConn) RemoteAddr() net.Addr {
	return c.realConn.RemoteAddr()
}

func (c *InterceptConn) SetDeadline(t time.Time) error {
	return c.realConn.SetDeadline(t)
}

func (c *InterceptConn) SetReadDeadline(t time.Time) error {
	return c.realConn.SetReadDeadline(t)
}

func (c *InterceptConn) SetWriteDeadline(t time.Time) error {
	return c.realConn.SetWriteDeadline(t)
}

func Handle(w http.ResponseWriter, r *http.Request, conns *ConnMap) {
	// w.WriteHeader(200)
	// w.Write([]byte("Hello World"))

	_, ok := conns.Pop(r.RemoteAddr)
	if !ok {
		fmt.Printf("ERROR RemoteAddr %v not in Conns\n", r.RemoteAddr)
		return
	}
	// interceptConn, ok := conn.(*InterceptConn)
	// if !ok {
	// 	fmt.Printf("ERROR Could not get Conn info: Conn is not an InterceptConn: %T\n", conn)
	// 	return
	// }
	// fmt.Printf("read %v bytes, wrote %v bytes\n", interceptConn.BytesRead(), interceptConn.BytesWritten())
}

// func ListenAndServeTLS(addr, certFile, keyFile string, proxy *Proxy) (*http.Server, error) {
// 	httpsListener, httpsConns, err := InterceptListenTLS("tcp", addr, certFile, keyFile)
// 	if err != nil || proxy == nil {
// 		return nil, err
// 	}

// 	newHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		proxy.ServeHTTP(w, r, httpsConns)
// 	})

// 	server := &http.Server{Handler: newHandler, Addr: addr}
// 	return server, server.Serve(httpsListener)
// }

// func ListenAndServe(addr string, proxy *Proxy) (*http.Server, error) {
// 	httpsListener, httpsConns, err := InterceptListen("tcp", addr)
// 	if err != nil || proxy == nil {
// 		return nil, err
// 	}

// 	newHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		proxy.ServeHTTP(w, r, httpsConns)
// 	})

// 	server := &http.Server{Handler: newHandler, Addr: addr}
// 	return server, server.Serve(httpsListener)
// }
