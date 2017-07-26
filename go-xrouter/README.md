# go-xrouter

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xrouter** is a trie based HTTP request router. Its implementation is based on [httprouter]()    
package, but you can delete an existing handler without creating a new route, or add a new    
handler in runtime. I think it's useful in some scenarios which you want to modify route dynamically.     

## Implementation

The main structure ([*Radix Tree*][Radix Tree]) of **go-xrouter** is same as **httprouter**, so if you  
want to know how it works, you can see the [Wiki][how_work] of **httprouter**. I just list some main   
differences between **go-xrouter** and **httprouter**:


- The configure options of *xrouter* are not exported, you only can set it when you create a   
`XRouter`. Because a `XRouter` is used by multiple goroutines, change the configure options     
directly is not concurrent safe (We shouldn't assume that **load/store** a bool or a function field   
is atomicity), so I hide them.
- Search a handler for a request, add a new handler or remove an existing handler, these three     
operations are restricted by [RWMutex](https://golang.org/pkg/sync/#RWMutex).
- If add a new handler failed, it will return a error describing the reason rather than leading the   
program panic.     
- Add `Remove` function to remove an existing handler.   


## Usage

```go

package main

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xrouter"
	"log"
	"net/http"
)

var xr *xrouter.XRouter

func Index(w http.ResponseWriter, r *http.Request, _ xrouter.XParams) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, xps xrouter.XParams) {
	fmt.Fprintf(w, "Hello, %s!\n", xps.Get("name"))
}

func Add(w http.ResponseWriter, r *http.Request, xps xrouter.XParams) {
	method, path := xps.Get("method"), "/"+xps.Get("path")
	err := xr.Handle(method, path, func(w http.ResponseWriter, r *http.Request, xps xrouter.XParams) {
		fmt.Fprintf(w, "path: %s, xps: %s\n", path, xps)
	})
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "[ERROR]: %s\n", err)
	} else {
		fmt.Fprintf(w, "OK\n")
	}
	return
}

func Remove(w http.ResponseWriter, r *http.Request, xps xrouter.XParams) {
	method, path := xps.Get("method"), "/"+xps.Get("path")
	err := xr.Remove(method, path)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "[ERROR]: %s\n", err)
	} else {
		fmt.Fprintf(w, "OK\n")
	}
}

func main() {
	xr = xrouter.New(&xrouter.XConfig{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleOptions:          true,
		HandleMethodNotAllowed: true,
	})

	xr.GET("/", Index)
	xr.GET("/hello/:name", Hello)
	xr.GET("/add/:method/*path", Add)
	xr.GET("/remove/:method/*path", Remove)

	log.Fatal(http.ListenAndServe(":8080", xr))
}
```

This demo supports add and remove a handler dynamically.

```bash
$ curl "http://127.0.0.1:8080/"
Welcome!

$ curl "http://127.0.0.1:8080/hello/blinklv"
Hello, blinklv!

$ curl "http://127.0.0.1:8080/add/GET/hello/:name"
[ERROR]: path '/hello/:name' has already been registered

$ curl "http://127.0.0.1:8080/add/POST/foo/:bar"
OK
$ curl -X POST "http://127.0.0.1:8080/foo/blinklv"
path: /foo/:bar, xps: bar=blinklv

$ curl "http://127.0.0.1:8080/remove/POST/foo/:bar"
OK
$ curl -X POST "http://127.0.0.1:8080/foo/blinklv"
404 page not found
```


[httprouter]: https://github.com/julienschmidt/httprouter
[how_work]: https://github.com/julienschmidt/httprouter#how-does-it-work
[Radix Tree]: https://en.wikipedia.org/wiki/Radix_tree

