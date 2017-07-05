// main.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-07-05
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-05
// Purpose: This program is used to test the validity of the
// XRouter (server end).

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xrouter"
	"net/http"
	"os"
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
	Port       string          `json:"port"`
	AdminPort  string          `json:"admin_port"`
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

	xr := xrouter.New(&cfg.XRouter)
	for _, p := range cfg.Paths {
		if err := handle(xr, p.Method, p.Path, generateHandle(p)); err != nil {
			return err
		}
	}

	for _, p := range cfg.PanicPaths {
		if err := handle(xr, p.Method, p.Path, generatePanicHandle(p)); err != nil {
			return err
		}
	}
}

func handle(xr *XRouter, method, path string, handle XHandle) error {
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

func generateHandle(p path) {
	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
		w.Write(newResponse(p))
	}
}

func generatePanicHandle(p path) {
	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
		panic(string(newResponse(p)))
	}
}

func newResponse(p path) []byte {
	var obj = struct {
		Method  string `json:"method"`
		Path    string `json:"path"`
		XParams string `json:"xparams"`
	}{
		Method:  p.Method,
		Path:    p.Path,
		XParams: xps.String(),
	}

	msg, _ := json.Marshal(obj)
	return msg
}

func addHandle(w http.ResponseWriter, r *http.Request) {
}

func removeHandle(w http.ResponseWriter, r *http.Request) {
}

// Adding a new path.
func add() error {
}

// Removing an existing path.
func remove() error {
}
