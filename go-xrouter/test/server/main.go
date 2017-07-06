// main.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-07-05
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-06
// Purpose: This program is used to test the validity of the
// XRouter (server end).

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xrouter"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	flagConfig    = flag.String("config", "./config.json", "configure file path")
	flagHost      = flag.String("host", "127.0.0.1", "broadcast host")
	flagPort      = flag.String("port", "9090", "http port listening for client")
	flagAdminPort = flag.String("admin-port", "9091", "http port listening for administrator")
	flagCmd       = flag.String("cmd", "run", "command (run, add, remove)")
	flagMethods   = flag.String("methods", "ALL", "http method, eg: GET,POST,DELETE. ALL represent all methods")
	flagPattern   = flag.String("path", "", "path path, eg: /who/:are/*you")
)

type config struct {
	XRouter    xrouter.XConfig `json:"xrouter"`
	Paths      []path          `json:"paths"`
	PanicPaths []path          `json:"panic_paths"`
}

type path struct {
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
}

func main() {
	flag.Parse()

	var err error

	switch *flagCmd {
	case "run":
		err = run()
	case "add":
		err = add()
	case "remove":
		err = remove()
	default:
		fmt.Fprintf(os.Stderr, "[ERROR]: Invalid command (%s)\n", *flagCmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: %s\n", err)
		os.Exit(1)
	}
}

var xr *xrouter.XRouter

// Run http server.
func run() error {
	data, err := ioutil.ReadFile(*flagConfig)
	if err != nil {
		return err
	}

	cfg := &config{}
	if err = json.Unmarshal(data, cfg); err != nil {
		return err
	}

	cfg.XRouter.PanicHandler = func(w http.ResponseWriter, r *http.Request, x interface{}) {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%s", x)))
	}

	xr = xrouter.New(&cfg.XRouter)
	for _, p := range cfg.Paths {
		for _, method := range p.Methods {
			if err := handle(xr, method, p.Path, generateHandle(method, p.Path)); err != nil {
				return err
			}
		}
	}

	for _, p := range cfg.PanicPaths {
		for _, method := range p.Methods {
			if err := handle(xr, method, p.Path, generatePanicHandle(method, p.Path)); err != nil {
				return err
			}
		}
	}

	// Run the HTTP server for the requests of the client.
	s := &http.Server{
		Addr:           ":" + *flagPort,
		Handler:        xr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go s.ListenAndServe()

	sm := http.NewServeMux()
	sm.HandleFunc("/add", generateAdminHandle("add"))
	sm.HandleFunc("/remove", generateAdminHandle("remove"))

	// Run the HTTP server for the requests of the administrator.
	as := &http.Server{
		Addr:           ":" + *flagAdminPort,
		Handler:        xr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	as.ListenAndServe()

	return nil
}

func handle(xr *xrouter.XRouter, method, path string, handle xrouter.XHandle) error {
	// Using 'Handle' function directly will be better, but I use this
	// 'handle' function to check the validity of the shortcut for 'Handle'.
	switch method {
	case "GET":
		return xr.GET(path, handle)
	case "POST":
		return xr.POST(path, handle)
	case "HEAD":
		return xr.HEAD(path, handle)
	case "PUT":
		return xr.PUT(path, handle)
	case "OPTIONS":
		return xr.OPTIONS(path, handle)
	case "PATCH":
		return xr.PATCH(path, handle)
	case "DELETE":
		return xr.DELETE(path, handle)
	default:
		return xr.Handle(method, path, handle)
	}
}

func generateHandle(method, p string) xrouter.XHandle {
	return func(w http.ResponseWriter, _ *http.Request, xps xrouter.XParams) {
		w.Write(newResponse(method, p, xps))
	}
}

func generatePanicHandle(method, p string) xrouter.XHandle {
	return func(w http.ResponseWriter, _ *http.Request, xps xrouter.XParams) {
		panic(string(newResponse(method, p, xps)))
	}
}

type response struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	XParams string `json:"xparams"`
}

func newResponse(method, p string, xps xrouter.XParams) []byte {
	var obj = &response{
		Method:  method,
		Path:    p,
		XParams: xps.String(),
	}

	msg, _ := json.Marshal(obj)
	return msg
}

type adminRequest struct {
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
}

type adminResponse struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func generateAdminHandle(cmd string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var arsp = &adminResponse{}

		defer func() {
			if arsp.Ret == -1 {
				w.WriteHeader(500)
			} else if arsp.Ret == -2 {
				w.WriteHeader(400)
			}

			data, _ := json.Marshal(arsp)
			w.Write(data)
		}()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			arsp.Ret, arsp.Msg = -1, err.Error()
			return
		}

		areq := &adminRequest{}
		if err = json.Unmarshal(body, areq); err != nil {
			arsp.Ret, arsp.Msg = -2, err.Error()
			return
		}

		// If 'Methods' contains 'ALL', ignore other methods.
		existAll := false
		for _, method := range areq.Methods {
			if strings.ToUpper(method) == "ALL" {
				existAll = true
				break
			}
		}

		var methods []string
		if existAll {
			methods = []string{"GET", "POST", "HEAD", "PUT", "OPTIONS", "PATCH", "DELETE"}
		} else {
			methods = areq.Methods
		}

		for _, method := range methods {
			if cmd == "add" {
				err = handle(xr, method, areq.Path, generateHandle(method, areq.Path))
			} else if cmd == "remove" {
				err = xr.Remove(method, areq.Path)
			}

			if err != nil {
				arsp.Ret, arsp.Msg = -2, err.Error()
				return
			}
		}

		return
	}
}

// Adding a new path.
func add() error {
	return nil
}

// Removing an existing path.
func remove() error {
	return nil
}
