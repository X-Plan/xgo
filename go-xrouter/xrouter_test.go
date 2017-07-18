// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-18
package xrouter

import (
	"encoding/json"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestXParam(t *testing.T) {
	xp1 := XParam{Key: "Hello", Value: "World"}
	xassert.Equal(t, fmt.Sprintf("%s", xp1), "Hello=World")
	xp2 := XParam{Key: "Foo", Value: "Bar"}
	xassert.Equal(t, fmt.Sprintf("%s", xp2), "Foo=Bar")
	xassert.IsFalse(t, xp1.Equal(xp2))
	xp3 := xp1
	xassert.IsTrue(t, xp1.Equal(xp3))
}

func TestXParams(t *testing.T) {
	xps1 := XParams{
		{"Hello", "World"},
		{"Who", "Are"},
		{"Foo", "Bar"},
	}

	xassert.Equal(t, fmt.Sprintf("%s", xps1), "Hello=World,Who=Are,Foo=Bar")
	for _, xp := range xps1 {
		xassert.Equal(t, xps1.Get(xp.Key), xp.Value)
	}
	xassert.Equal(t, xps1.Get("World"), "")

	xps2 := xps1
	xassert.IsTrue(t, xps1.Equal(xps2))

	// The order of the elements should be same.
	xps3 := make(XParams, len(xps1))
	copy(xps3, xps1)
	xps3[0], xps3[1] = xps3[1], xps3[0]
	xassert.IsFalse(t, xps1.Equal(xps3))

	xps4 := xps1[0 : len(xps1)-1]
	xassert.IsFalse(t, xps1.Equal(xps4))

	xps5 := append(xps1, XParam{"A", "B"})
	xassert.IsFalse(t, xps1.Equal(xps5))
}

func TestSupportMethod(t *testing.T) {
	var methodPair = []struct {
		method string
		ok     bool
	}{
		{"get", true},
		{"pOst", true},
		{"HEAD", true},
		{"puT", true},
		{"options", true},
		{"PATCh", true},
		{"DELETE", true},
		{"CONNECT", false},
		{"foo", false},
		{"BAR", false},
	}

	for _, pair := range methodPair {
		xassert.Equal(t, SupportMethod(pair.method), pair.ok)
	}
}

func TestHandle(t *testing.T) {
	var paths = []struct {
		methods []string
		path    string
		ok      bool
	}{
		{[]string{"GET", "POST"}, "/hello/:world", true},
		{[]string{"get"}, "/hello/:world", false},        // has been existing
		{[]string{"post"}, "/hello/world", false},        // conflict
		{[]string{"GET", "POST"}, "hello/:world", false}, // path doesn't begin with '/'
		{[]string{"GET", "POST"}, "", false},             // path doesn't begin with '/'
		{[]string{"PUT", "OPTIONS", "DELETE"}, "/hello/:world", true},
		{[]string{"CONNECT"}, "/foo/bar", false}, // method is unsupported
	}

	xr := New(&XConfig{})
	for _, p := range paths {
		var err error
		for _, method := range p.methods {
			if err = xr.Handle(method, p.path, generateHandle(method, p.path)); err != nil {
				break
			}
		}
		xassert.Equal(t, err == nil, p.ok)
	}
}

func TestPanic(t *testing.T) {
	xr := New(&XConfig{
		PanicHandler: func(w http.ResponseWriter, r *http.Request, x interface{}) {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", x)))
		},
	})

	method, path := "GET", "/I/am/panic"
	xassert.IsNil(t, handle(xr, method, path, generatePanicHandle(method, path)))
	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()
	path, xps := generatePath(path)
	xassert.IsNil(t, roundtrip(port, method, path, xps, 500, check200_and_500(method, path, xps)))
}

func runServer(xr *XRouter) (l net.Listener, port string, err error) {
	if l, err = net.Listen("tcp", "0.0.0.0:0"); err != nil {
		return
	}
	_, port, _ = net.SplitHostPort(l.Addr().String())

	s := &http.Server{
		Handler:        xr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go s.Serve(l)
	return
}

type checkFunc func(*http.Response) error

func roundtrip(port, method, path string, xps XParams, code int, check checkFunc) error {
	req, err := http.NewRequest(method, "http://127.0.0.1:"+port+path, nil)
	if err != nil {
		return err
	}

	rsp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}

	if rsp.StatusCode != code {
		return fmt.Errorf("response status code (%d) is not equal to expected status code (%d)", rsp.StatusCode, code)
	}

	return check(rsp)
}

func check200_and_500(method, path string, xps XParams) checkFunc {
	return func(rsp *http.Response) error {
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}

		pkg := &response{}
		if err = json.Unmarshal(body, pkg); err != nil {
			return err
		}

		if pkg.Method != method {
			return fmt.Errorf("response method (%s) is not equal to expected method (%s)", pkg.Method, method)
		}

		if len(xps) == 0 {
			// static path.
			if pkg.Path != path {
				return fmt.Errorf("response path (%s) is not equal to expected path (%s)", pkg.Path, path)
			}
		} else {
			// param path.
			if pkg.XParams != xps.String() {
				return fmt.Errorf("response params (%s) is not equal to expected params (%s)", pkg.XParams, xps)
			}
		}

		return nil
	}
}

func check301_and_307(redirectPath string) checkFunc {
	return func(rsp *http.Response) error {
		if rsp.Header.Get("Location") != redirectPath {
			return fmt.Errorf("response location (%s) is not equal to expected path (%s)", rsp.Header.Get("Location"), redirectPath)
		}
		return nil
	}
}

// Handle a path on multiple methods.
func mhandle(xr *XRouter, methods []string, path string, h XHandle) (err error) {
	for _, method := range methods {
		if err = handle(xr, method, path, h); err != nil {
			return
		}
	}
	return
}

// Using 'Handle' function directly will be better, but I use this
// 'handle' function to check the validity of the shortcut for 'Handle'.
func handle(xr *XRouter, method, path string, h XHandle) error {
	switch method {
	case "GET":
		return xr.GET(path, h)
	case "POST":
		return xr.POST(path, h)
	case "HEAD":
		return xr.HEAD(path, h)
	case "PUT":
		return xr.PUT(path, h)
	case "OPTIONS":
		return xr.OPTIONS(path, h)
	case "PATCH":
		return xr.PATCH(path, h)
	case "DELETE":
		return xr.DELETE(path, h)
	default:
		return xr.Handle(method, path, h)
	}
}

type response struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	XParams string `json:"xparams"`
}

func newResponse(method, p string, xps XParams) []byte {
	var obj = &response{
		Method:  method,
		Path:    p,
		XParams: xps.String(),
	}

	msg, _ := json.Marshal(obj)
	return msg
}

func generateHandle(method, p string) XHandle {
	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
		w.Write(newResponse(method, p, xps))
	}
}

func generatePanicHandle(method, p string) XHandle {
	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
		panic(string(newResponse(method, p, xps)))
	}
}
