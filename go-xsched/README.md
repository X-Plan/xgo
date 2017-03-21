# go-xsched

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xsched** is load balancing scheduler, its implementation is based on *[Weight Round-Robin]* 
algorithm.

### Usage


``` go

package main

import (
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xsched"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
)

type Transport struct {
	rt http.RoundTripper
	xs *xsched.XScheduler
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rsp, err := t.rt.RoundTrip(req)
	if err != nil || rsp.StatusCode/100 == 5 {
		// Feedback failure when 'RoundTrip' function returns a error or backend
		// service returns 5xx status code, otherwise feedback success.
		t.xs.Feedback(req.Host, false)
	} else {
		t.xs.Feedback(req.Host, true)
	}
	return rsp, err
}

// Create n backend test servers.
func createBackendServer(n int) (bs []*httptest.Server) {
	for i := 0; i < n; i++ {
		bs = append(bs, func(i int, src rand.Source) *httptest.Server {
			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Sometimes return 500.
				result := !(src.Int63()%10 == 0)
				if !result {
					w.WriteHeader(500)
				}
				fmt.Fprintf(w, "I'm backend server %d (result: %v)\n", i, result)
			}))
		}(i, rand.NewSource(int64(1000*i))))
	}
	return
}

func extractAddress(bs []*httptest.Server) (strs []string) {
	for i, b := range bs {
		strs = append(strs, b.URL[7:]+":"+strconv.Itoa(i%5+1))
	}
	return strs
}

var flagNumber = flag.Int("number", 5, "number of backend server")

func main() {
	flag.Parse()
	bs := createBackendServer(*flagNumber)
	defer func() {
		for _, b := range bs {
			b.Close()
		}
	}()

	xs, _ := xsched.New(extractAddress(bs))

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			address, _ := xs.Get()
			// Change the destination host of the request, this field
			// also be used in 'Transport.RoundTrip' method.
			r.URL.Scheme = "http"
			r.URL.Host = address
		},
		Transport: &Transport{http.DefaultTransport, xs},
	}

	http.ListenAndServe(":9192", proxy)
}

```


[Weight Round-Robin]: http://techcodecorner.blogspot.com/2014/03/the-weighted-round-robin-schedulingis.html
