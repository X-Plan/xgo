// xrouter.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-02-27
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-27

// Package go-xrouter is a trie based HTTP request router.
//
// Its implementation is based on 'github.com/julienschmidt/httprouter' package,
// but you can delete an existing handler without creating a new route. I think
// it's useful in some scenarios which you want to  modify route dynamically.
package xrouter

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// XHandle is a function that can be registered to a route to handle HTTP
// requests. Like http.HandleFunc, but has a third parameter for the values
// of wildcards.
type XHandle func(http.ResponseWriter, *http.Request, XParams)

// XParam is a key-value pair representing a single URL parameter.
type XParam struct {
	Key   string
	Value string
}

// The string format of a XParam is 'key=value'.
func (xp XParam) String() string {
	return xp.Key + "=" + xp.Value
}

// When two 'XParam's equal，the key and value of them must be equal.
func (xp XParam) Equal(a XParam) bool {
	return xp.Key == a.Key && xp.Value == a.Value
}

// XParams is a XParam-Slice, which returned by the XRouter.
// The slice is ordered so you can safely read values by the index.
type XParams []XParam

// The string format of a XParams is 'key1=value1,key2=value2,key3=value3'.
func (xps XParams) String() string {
	var str = ""
	for i, xp := range xps {
		if i > 0 {
			str += "," + xp.String()
		} else {
			str = xp.String()
		}
	}
	return str
}

// Get function returns the value of the first XParam which key matches the
// given name. If no matching XParam is found, it returns an empty string.
//
// NOTE: The length of XParams is small in most cases, so linear search is
// enough regarding efficiency.
func (xps XParams) Get(name string) string {
	for _, xp := range xps {
		if xp.Key == name {
			return xp.Value
		}
	}
	return ""
}

// When two 'XParams's are equal, they should meet the following conditions.
// 1. XParams1.length = XParams2.length
// 2. For each i between 0 and XParams1(2).length (exclude it): XParams1(i) = XParams2(i)
// One special case that two 'XParams's are empty, returns true.
func (xps XParams) Equal(as XParams) (equal bool) {
	if len(xps) == len(as) {
		if len(xps) == 0 {
			equal = true
			return
		}

		for i, xp := range xps {
			if equal = xp.Equal(as[i]); !equal {
				break
			}
		}
	}
	return
}

// This function is used to set the 'MethodNotAllowed' field of the 'XRouter'
// when you don't set it, you should covert it to 'http.HandlerFunc' type.
func DefaultMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(405), 405)
}

var methods = []string{"GET", "POST", "HEAD", "PUT", "OPTIONS", "PATCH", "DELETE"}

// This function is used to check whether the http method is supported by XRouter.
func SupportMethod(method string) bool {
	for _, m := range methods {
		if strings.ToUpper(method) == m {
			return true
		}
	}
	return false
}

// XConfig is used to create a new XRouter.
type XConfig struct {
	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router will reply to OPTIONS requests, but
	// the custom OPTIONS handlers has more priority than automatic replies.
	HandleOptions bool

	// If the current request can't be routed, it will check whether another
	// method is allowed for the current request when this option is enabled.
	// If other method has router to handle this request, will invoke the
	// MethodNotAllowed handler to response it, otherwise the request is
	// delegated to the NotFound handler.
	HandleMethodNotAllowed bool

	// When the request url path is not matching any register route, NotFound
	// handler will be called, If it's not set, http.NotFound is used.
	NotFound http.Handler

	// Whe the request method is not matching any register route, MethodNotAllowed
	// handler will be called. If it's not set, the DefaultMethodNotAllowed is
	// used (Its implementation is just wrapping 'http.Error(w, "Method Not Allowed", 405)').
	MethodNotAllowed http.Handler

	// Function to handle panics recovered from http handlers.The handler can be
	// used to keep your server from crashing because of unrecovered panics. You
	// should return the http error code 500 (Internal Server Error) in this handler.
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}

// XRouter is the implementation of the 'http.Handler', which can be
// dispatch requests to different handler functions via register routes.
type XRouter struct {
	trees map[string]*tree

	// The following fields are same as the fields in XConfig
	// except the first letter is lowercase. Because XRouter
	// is designed to run safely in a concurrent environment,
	// so the fields of XRouter can't be exported to user.
	redirectTrailingSlash  bool
	redirectFixedPath      bool
	handleOptions          bool
	handleMethodNotAllowed bool
	notFound               http.Handler
	methodNotAllowed       http.Handler
	panicHandler           func(http.ResponseWriter, *http.Request, interface{})
}

// New returns a new initialized XRouter.
func New(xcfg *XConfig) *XRouter {
	if xcfg == nil {
		return nil
	}

	xr := &XRouter{
		trees: make(map[string]*tree),
		redirectTrailingSlash:  xcfg.RedirectTrailingSlash,
		redirectFixedPath:      xcfg.RedirectFixedPath,
		handleOptions:          xcfg.HandleOptions,
		handleMethodNotAllowed: xcfg.HandleMethodNotAllowed,
		notFound:               xcfg.NotFound,
		methodNotAllowed:       xcfg.MethodNotAllowed,
		panicHandler:           xcfg.PanicHandler,
	}

	if xr.notFound == nil {
		xr.notFound = http.HandlerFunc(http.NotFound)
	}

	if xr.methodNotAllowed == nil {
		xr.methodNotAllowed = http.HandlerFunc(DefaultMethodNotAllowed)
	}

	for _, method := range methods {
		xr.trees[method] = &tree{&sync.RWMutex{}, &node{}}
	}

	return xr
}

// GET is a shortcut for Handle("GET", path, handle)
func (xr *XRouter) GET(path string, handle XHandle) error {
	return xr.Handle("GET", path, handle)
}

// POST is a shortcut for Handle("POST", path, handle)
func (xr *XRouter) POST(path string, handle XHandle) error {
	return xr.Handle("POST", path, handle)
}

// HEAD is a shortcut for Handle("HEAD", path, handle)
func (xr *XRouter) HEAD(path string, handle XHandle) error {
	return xr.Handle("HEAD", path, handle)
}

// PUT is a shortcut for Handle("PUT", path, handle)
func (xr *XRouter) PUT(path string, handle XHandle) error {
	return xr.Handle("PUT", path, handle)
}

// OPTIONS is a shortcut for Handle("OPTIONS", path, handle)
func (xr *XRouter) OPTIONS(path string, handle XHandle) error {
	return xr.Handle("OPTIONS", path, handle)
}

// PATCH is a shortcut for Handle("PATCH", path, handle)
func (xr *XRouter) PATCH(path string, handle XHandle) error {
	return xr.Handle("PATCH", path, handle)
}

// DELETE is a shortcut for Handle("DELETE", path, handle)
func (xr *XRouter) DELETE(path string, handle XHandle) error {
	return xr.Handle("DELETE", path, handle)
}

// Handle registers a new request handle with the given path and method.
// The registered path, against which the router matches incoming requests,
// can contain two types of parameters:
//  Syntax      Type
//  :name       named parameter
//  *name       catch-all parameter
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//  Path: /blog/:category/:post
//
//  Requests:
//   /blog/go/request-routers            match: category="go", post="request-routers"
//   /blog/go/request-routers/           no match, but the router would redirect
//   /blog/go/                           no match
//   /blog/go/request-routers/comments   no match
//
// Catch-all parameters match anything until the path end, including the
// directory index (the '/' before the catch-all). Since they match anything
// until the end, catch-all parameters must always be the final path element.
//  Path: /files/*filepath
//
//  Requests:
//   /files/                             match: filepath="/"
//   /files/LICENSE                      match: filepath="/LICENSE"
//   /files/templates/article.html       match: filepath="/templates/article.html"
//   /files                              no match, but the router would redirect
//
// The value of parameters is saved as a XParams, consisting each of a key and
// a value. The XParams is passed to the Handle func as a third parameter.
// There are two ways to retrieve the value of a parameter:
//  1. By the name of the parameter
//  user := xps.Get("user") // defined by :user or *user
//
//  2. By the index of the parameter.
//  thirdKey    := xps[2].Key     // the name of the 3rd parameter
//  thirdValue  := xps[2].Value   // the value of the 3rd parameter
//
func (xr *XRouter) Handle(method, path string, handle XHandle) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("path ('%s') must begin with '/'", path)
	}

	t := xr.trees[strings.ToUpper(method)]
	if t == nil {
		return fmt.Errorf("http method (%s) is unsupported", method)
	}

	return t.add(path, handle)
}

// Remove unregister a existing request handle with given path and method.
// The path parameter must match a existing path exactly, It won't consider
// wildcard and trailing slash. If the path '/foo/:bar/who/are/*you' has
// been registered, we can't use the argument '/foo/:xxx/who/are/*ooo' to
// remove it.
func (xr *XRouter) Remove(method, path string) error {
	t := xr.trees[strings.ToUpper(method)]
	if t == nil {
		return fmt.Errorf("http method (%s) is unsupported", method)
	}

	return t.remove(path)
}

// ServeHTTP is the implementation of the http.Handler interface.
func (xr *XRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if xr.panicHandler != nil {
		defer xr.capturePanic(w, r)
	}

	path, hasFixed := r.URL.Path, false

	if t := xr.trees[r.Method]; t != nil {

	fixed:
		// If the results of the t.isempty function equals to true,
		// the t.get function will also return the nil handle.
		if handle, xps, tsr := t.get(path, xr.redirectTrailingSlash); handle != nil {
			handle(w, r, xps)
			return
		} else if r.Method != "CONNECT" && path != "/" {
			code := 301 // Permanent redirect, request with GET method
			if r.Method != "GET" {
				// Temporary redirect, request with same method
				// As of Go 1.3, Go does not support status code 308.
				code = 307
			}

			if xr.redirectTrailingSlash && tsr != notRedirect {
				r.URL.Path = xr.redirectPath(path, tsr)
				http.Redirect(w, r, r.URL.String(), code)
				return
			}

			// Try to fix the request path. The 'hasFixed' variable can be
			// ignored if don't consider performance, because the 'fixedPath'
			// must equal to 'CleanPath(path)' on second time.
			if xr.redirectFixedPath && !hasFixed {
				fixedPath := CleanPath(path)
				if fixedPath != path {
					path, hasFixed = fixedPath, true
					goto fixed
				}
			}
		}
	}

	if r.Method == "OPTIONS" {
		if xr.handleOptions {
			// Handle OPTIONS requests.
			if allow := xr.allowed(r.URL.Path, r.Method); len(allow) > 0 {
				w.Header().Set("Allow", allow)
				return
			}
		}
	} else {
		// Handle 405.
		if xr.handleMethodNotAllowed {
			if allow := xr.allowed(r.URL.Path, r.Method); len(allow) > 0 {
				w.Header().Set("Allow", allow)
				xr.methodNotAllowed.ServeHTTP(w, r)
				return
			}
		}
	}

	// Other case returns 404.
	xr.notFound.ServeHTTP(w, r)
}

func (xr *XRouter) allowed(path, reqMethod string) (allow string) {
	if path == "*" && reqMethod == "OPTIONS" {
		for method, t := range xr.trees {
			if method == "OPTIONS" || t.isempty() {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else {
		for method, t := range xr.trees {
			if method == reqMethod || method == "OPTIONS" || t.isempty() {
				continue
			}

			if handle, _, tsr := t.get(path, xr.redirectTrailingSlash); handle != nil || tsr > 0 {
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}

	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return
}

func (xr *XRouter) redirectPath(path string, tsr tsrType) string {
	if tsr == addSlash {
		return path + "/"
	} else if tsr == removeSlash {
		return path[:len(path)-1]
	}
	return path
}

func (xr *XRouter) capturePanic(w http.ResponseWriter, r *http.Request) {
	if x := recover(); x != nil {
		xr.panicHandler(w, r, x)
	}
}
