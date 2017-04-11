// xrouter.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-02-27
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-11

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
	// If the current route can't be matched, but a handler for the
	// path with (without) the trailing slash exists, which will be
	// used to handle this request. For example if the path of request
	// is /foo/,  but a route only exists for /foo, the handler of
	// /foo will be used.
	CompatibleWithTrailingSlash bool

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
	// so the fields of XRouter can't export to user.
	compatibleWithTrailingSlash bool
	handleOptions               bool
	handleMethodNotAllowed      bool
	notFound                    http.Handler
	methodNotAllowed            http.Handler
	panicHandler                func(http.ResponseWriter, *http.Request, interface{})
}

// New returns a new initialized XRouter. All options is enabled by default.
func New(xcfg *XConfig) *XRouter {
	xr := &XRouter{
		trees: make(map[string]*tree),
		compatibleWithTrailingSlash: xcfg.CompatibleWithTrailingSlash,
		handleOptions:               xcfg.HandleOptions,
		handleMethodNotAllowed:      xcfg.HandleMethodNotAllowed,
		notFound:                    xcfg.NotFound,
		methodNotAllowed:            xcfg.MethodNotAllowed,
		panicHandler:                xcfg.PanicHandler,
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

// Handle registers a new request handle with the given path and method.
func (xr *XRouter) Handle(method, path string, handle XHandle) error {
	t := xr.trees[method]
	if t == nil {
		return fmt.Errorf("http method (%s) is unsupported", method)
	}

	// Fixing the path before it's registered.
	return t.add(CleanPath(path), handle)
}

// ServeHTTP is the implementation of the http.Handler interface.
func (xr *XRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if xr.panicHandler != nil {
		defer xr.capturePanic(w, r)
	}

	// Fix the current request path, but exclude the asterisk ('*').
	path := "*"
	if r.URL.Path != "*" {
		path = CleanPath(r.URL.Path)
	}

	var xps XParams

	if t := xr.trees[r.Method]; t != nil {
		// If the results of the t.isempty function equals to true,
		// the t.get function will also return the nil.
		if handle := t.get(path, &xps, xr.compatibleWithTrailingSlash); handle != nil {
			handle(w, r, xps)
			return
		}
	}

	if r.Method == "OPTIONS" {
		// Handle OPTIONS requests.
		if allow := xr.allowed(path, r.Method); len(allow) > 0 {
			w.Header().Set("Allow", allow)
			return
		}
	} else {
		// Handle 405.
		if xr.handleMethodNotAllowed {
			if allow := xr.allowed(path, r.Method); len(allow) > 0 {
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

			var xps XParams
			if t.get(path, &xps, xr.compatibleWithTrailingSlash) != nil {
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

func (xr *XRouter) capturePanic(w http.ResponseWriter, r *http.Request) {
	if x := recover(); x != nil {
		xr.panicHandler(w, r, x)
	}
}
