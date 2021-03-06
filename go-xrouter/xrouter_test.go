// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-25
package xrouter

import (
	"encoding/json"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strings"
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

	paths := []pathType{
		{[]string{"GET", "POST"}, "/I/am/panic", nil},
		{[]string{"PUT", "DELETE"}, "/hello/:world", nil},
		{[]string{"GET"}, "/who/a:re/*you", nil},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generatePanicHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	for _, p := range paths {
		for _, method := range p.methods {
			path, xps := generatePath(p.path)
			xassert.IsNil(t, roundtrip(port, method, path, xps, 500, check200_and_500(method, path, xps)))
		}
	}
}

func TestRedirectTrailingSlash(t *testing.T) {
	xr := New(&XConfig{RedirectTrailingSlash: true})
	paths := []pathType{
		{[]string{"GET", "POST"}, "/get/user/info", addSlash},
		{[]string{"GET", "POST"}, "/get/user/foo/", removeSlash},
		{[]string{"PUT"}, "/admin/add/:user/", removeSlash},
		{[]string{"DELETE"}, "/admin/del/:user", addSlash},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generateHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	for _, p := range paths {
		for _, method := range p.methods {
			path, xps := generatePath(p.path)
			code := 301
			if method != "GET" {
				code = 307
			}

			redirectPath := path
			if p.ext.(tsrType) == removeSlash {
				path = path[:len(path)-1]
			} else if p.ext.(tsrType) == addSlash {
				path += "/"
			}

			xassert.IsNil(t, roundtrip(port, method, path, xps, code, check301_and_307(redirectPath)))
		}
	}
}

func TestRedirectFixedPath(t *testing.T) {
	xr := New(&XConfig{RedirectFixedPath: true})
	paths := []pathType{
		{[]string{"GET"}, "/get/user/info", "/get////user/info"},
		{[]string{"POST", "PUT"}, "/hello/:world/", "/../../////hello///:world///"},
		{[]string{"GET", "DELETE"}, "/who/are/", "/who/are/./"},
		{[]string{"GET", "DELETE"}, "/who", "/who/are/.."},
		{[]string{"GET", "DELETE"}, "/who/", "/who/are/../"},
		{[]string{"POST", "PATCH"}, "/what/you/*want", "/what/./you/foo/../*want"},
		{[]string{"DELETE", "POST"}, "/for/:yourself/", "/../for/bar/.././:yourself////"},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generateHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	for _, p := range paths {
		for _, method := range p.methods {
			redirectPath, xps := generatePath(p.path)
			code := 301
			if method != "GET" {
				code = 307
			}

			path := p.ext.(string)
			for _, xp := range xps {
				path = strings.Replace(strings.Replace(path, ":"+xp.Key, xp.Value, -1), "*"+xp.Key, xp.Value, -1)
			}

			xassert.IsNil(t, roundtrip(port, method, path, xps, code, check301_and_307(redirectPath)))
		}
	}
}

func TestHandleMethodNotAllowed(t *testing.T) {
	xr := New(&XConfig{HandleMethodNotAllowed: true})
	paths := []pathType{
		{[]string{"GET", "POST", "PATCH", "OPTIONS"}, "/hello/world", nil},
		{[]string{"PUT", "GET", "POST"}, "/get/user/:info", nil},
		{[]string{"HEAD", "DELETE", "POST", "OPTIONS"}, "/add/user/*info", nil},
		{[]string{"HEAD", "GET", "DELETE"}, "/what/:you/want/for/:me", nil},
		{[]string{"GET", "DELETE"}, "/I/want/to/:go/*home", nil},
		{[]string{"PUT", "POST"}, "/dont/:give/up/", nil},
		{[]string{"OPTIONS"}, "/collect/user/:information/", nil},
		{[]string{"HEAD", "OPTIONS"}, "/get/out/", nil},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generateHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	for _, p := range paths {
		ms := diffMethods(methods, p.methods)
		for _, method := range ms {
			if method == "OPTIONS" {
				continue
			}
			path, xps := generatePath(p.path)
			allowed := append(diffMethods(p.methods, []string{"OPTIONS"}), "OPTIONS")
			xassert.IsNil(t, roundtrip(port, method, path, xps, 405, check405(allowed)))
		}
	}
}

func TestHandleOptions(t *testing.T) {
	xr := New(&XConfig{HandleOptions: true})
	paths := []pathType{
		{[]string{"GET", "POST"}, "/a/b/c", false},
		{[]string{"GET", "POST", "PUT"}, "/get/user/:information", false},
		{[]string{"POST", "DELETE"}, "/what/are/*you", false},
		{[]string{"OPTIONS"}, "/what/:you/want/for/*me", true},
		{[]string{"OPTIONS"}, "/hello/:world/", true},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generateHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	for _, p := range paths {
		method := "OPTIONS"
		path, xps := generatePath(p.path)
		if p.ext.(bool) {
			xassert.IsNil(t, roundtrip(port, method, path, xps, 200, check200_and_500(method, path, xps)))
		} else {
			allowed := append(diffMethods(p.methods, []string{"OPTIONS"}), "OPTIONS")
			xassert.IsNil(t, roundtrip(port, method, path, xps, 200, check405(allowed)))
		}
	}
}

func TestAll(t *testing.T) {
	xr := New(&XConfig{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleOptions:          true,
		HandleMethodNotAllowed: true,
	})
	paths := []pathType{
		{[]string{"GET", "POST"}, "/authorizations", nil},
		{[]string{"GET", "DELETE"}, "/authorizations/:id", nil},
		{[]string{"GET", "DELETE"}, "/applications/:client_id/tokens/:access_token", nil},
		{[]string{"DELETE"}, "/applications/:client_id/tokens", nil},
		{[]string{"GET"}, "/events", nil},
		{[]string{"GET"}, "/repos/:owner/:repo/events", nil},
		{[]string{"GET"}, "/networks/:owner/:repo/events", nil},
		{[]string{"POST"}, "/orgs/:org/events", nil},
		{[]string{"GET"}, "/users/:user/received_events", nil},
		{[]string{"GET"}, "/users/:user/received_events/public", nil},
		{[]string{"GET"}, "/users/:user/events", nil},
		{[]string{"GET"}, "/users/:user/events/public", nil},
		{[]string{"GET"}, "/users/:user/events/orgs/:org", nil},
		{[]string{"GET"}, "/feeds", nil},
		{[]string{"GET"}, "/notifications", nil},
		{[]string{"GET", "PUT"}, "/repos/:owner/:repo/notifications/", nil},
		{[]string{"PUT"}, "/notifications", nil},
		{[]string{"OPTIONS"}, "/notifications/threads/*id", nil},
	}
	xassert.IsNil(t, configureXRouter(xr, paths, generateHandle))

	l, port, err := runServer(xr)
	xassert.IsNil(t, err)
	defer l.Close()

	requestPaths := []pathType{
		{[]string{"GET", "DELETE"}, "/authorizations/123456", XParams{{"id", "123456"}}},
		{[]string{"GET", "DELETE"}, "/applications/123456/tokens/123456", XParams{{"client_id", "123456"}, {"access_token", "123456"}}},
		{[]string{"DELETE"}, "/applications/123456/tokens", XParams{{"client_id", "123456"}}},
		{[]string{"OPTIONS"}, "/notifications/threads/123456", XParams{{"id", "123456"}}},
		{[]string{"OPTIONS"}, "/authorizations", []string{"GET", "POST", "OPTIONS"}},
		{[]string{"GET"}, "/events/", "/events"},
		{[]string{"GET"}, "/users/blinklv/events/public/", "/users/blinklv/events/public"},
		{[]string{"GET", "PUT"}, "/repos/blinklv/go-xrouter/notifications", "/repos/blinklv/go-xrouter/notifications/"},
		{[]string{"GET"}, "/users/blinklv/events/public/.//////", "/users/blinklv/events/public"},
		{[]string{"POST"}, "/../../orgs/X-Plan/.////events///", "/orgs/X-Plan/events"},
		{[]string{"POST"}, "/feeds/", []string{"GET", "OPTIONS"}},
		{[]string{"DELETE"}, "/notifications/threads/123456", []string{"OPTIONS"}},
		{[]string{"POST"}, "/applications/123456/tokens/123456", []string{"GET", "DELETE", "OPTIONS"}},
	}

	for _, rpath := range requestPaths {
		path := rpath.path
		for _, method := range rpath.methods {
			switch ext := rpath.ext.(type) {
			case XParams:
				xassert.IsNil(t, roundtrip(port, method, path, ext, 200, check200_and_500(method, path, ext)))
			case string:
				code := 301
				if method != "GET" {
					code = 307
				}
				xassert.IsNil(t, roundtrip(port, method, path, XParams{}, code, check301_and_307(ext)))
			case []string:
				code := 405
				if method == "OPTIONS" {
					code = 200
				}
				xassert.IsNil(t, roundtrip(port, method, path, XParams{}, code, check405(ext)))
			}
		}
	}
}

type pathType struct {
	methods []string
	path    string
	ext     interface{}
}

func diffMethods(a, b []string) []string {
	var c []string
	for _, am := range a {
		var exist bool
		for _, bm := range b {
			if am == bm {
				exist = true
				break
			}
		}

		if !exist {
			c = append(c, am)
		}
	}

	return c
}

func configureXRouter(xr *XRouter, paths []pathType, generate func(string, string) XHandle) (err error) {
	for _, p := range paths {
		for _, method := range p.methods {
			if err = handle(xr, method, p.path, generate(method, p.path)); err != nil {
				return
			}
		}
	}

	return
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

	rsp, err := (&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(req)
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

func check404(*http.Response) error {
	return nil
}

func check405(allowed []string) checkFunc {
	return func(rsp *http.Response) error {
		allowHeader := strings.Split(rsp.Header.Get("Allow"), ", ")
		sort.Strings(allowed)
		sort.Strings(allowHeader)
		if !reflect.DeepEqual(allowed, allowHeader) {
			return fmt.Errorf("response Allow (%s) and expected Allow (%s) don't match", rsp.Header.Get("Allow"), strings.Join(allowed, ", "))
		}
		return nil
	}
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
