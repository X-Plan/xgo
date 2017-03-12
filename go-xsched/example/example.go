// example.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-12
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-12

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
				result := src.Int63()%10 == 0
				if result {
					w.WriteHeader(500)
				}
				fmt.Fprintln(w, "I'm backend server %d (result: %v)", i, result)
			}))
		}(i, rand.NewSource(int64(i))))
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
			r.Host = address
		},
		Transport: &Transport{http.DefaultTransport, xs},
	}

	http.ListenAndServe(":9192", proxy)
}
