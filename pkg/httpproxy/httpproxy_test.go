package httpproxy

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	// "github.com/acentior/infiniteproxies_backend/pkg/logging"

	"github.com/stretchr/testify/suite"
	// "go.uber.org/zap/zapcore"
)

type HandlerSuite struct {
	suite.Suite
}

func (s *HandlerSuite) SetupSuite() {
	// logging.SetLevel(zapcore.FatalLevel)
}

func (s *HandlerSuite) SetupTest() {
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) Test_proxyTest() {
	// Create a new proxy with default certificate pair.
	prx, err := NewProxy()
	if err != nil {
		log.Printf("Failed to init proxy server. error: %v", err)
	}

	// Set handlers.
	prx.OnError = func(ctx *Context, where string, err *Error, opErr error) {
		// Log errors.
		log.Printf("ERR: %s: %s [%s]", where, err, opErr)
	}
	prx.OnAccept = func(ctx *Context, w http.ResponseWriter, r *http.Request) bool {
		// Handle local request has path "/info"
		if r.Method == "GET" && !r.URL.IsAbs() && r.URL.Path == "/info" {
			w.Write([]byte("This is go-httpproxy."))
			return true
		}
		return false
	}
	prx.OnAuth = func(ctx *Context, authType, user, pass string) bool {
		// Auth test user.
		if user == "test" && pass == "1234" {
			return true
		}
		return false
	}
	prx.OnConnect = func(ctx *Context, host string) (ConnectAction ConnectAction, newHost string) {
		// Apply "Man in the Middle" to all ssl connections. Never change host.
		return ConnectMitm, host
	}
	prx.OnRequest = func(ctx *Context, req *http.Request) (resp *http.Response) {
		// Log proxying requests.
		log.Printf("INFO: Proxy: %s %s", req.Method, req.URL.String())
		return
	}
	prx.OnResponse = func(ctx *Context, req *http.Request, resp *http.Response) {
		// Add header "Via: go-httpproxy".
		resp.Header.Add("Via", "go-httpproxy")
	}

	// Listen...
	server := httptest.NewServer(prx)
	defer server.Close()

	// proxy server URL
	fmt.Println(server.URL)
	proxyUrl, err := url.Parse(server.URL)
	s.NoError(err)
	myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

	type testCase struct {
		name, url, proxyAuth string
		shouldFail           bool
	}

	tcs := []testCase{
		{
			name:      "TestExistingEndpoint",
			url:       "http://www.google.com",
			proxyAuth: "test:1234",
		},
		{
			name:       "TestMissingEndpoint",
			url:        "/badendpoint",
			proxyAuth:  "test:1234",
			shouldFail: true,
		},
	}

	for _, tc := range tcs {
		req, err := http.NewRequest("GET", tc.url, nil)
		s.NoError(err)

		// Add proxy auth header in the http request
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(tc.proxyAuth))
		req.Header.Add("Proxy-Authorization", basicAuth)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, err := myClient.Do(req)
		s.NoError(err)

		// Read body data
		defer resp.Body.Close()

		if !tc.shouldFail {
			body, err := ioutil.ReadAll(resp.Body)
			s.NoError(err)
			fmt.Printf("Body : %s", body)
			return
		}
	}
}
