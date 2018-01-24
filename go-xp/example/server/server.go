// server.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-12-19
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-25

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xlog"
	"github.com/X-Plan/xgo/go-xp"
	"github.com/X-Plan/xgo/go-xserver"
	"io/ioutil"
	"os"
)

type Config struct {
	Port     string          `json:"port"`
	Log      xlog.XConfig    `json:"log"`
	TLS      TLSConfig       `json:"tls"`
	Handlers []HandlerConfig `json:"handlers"`
}

type TLSConfig struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

type HandlerConfig struct {
	Cmd          int      `json:"cmd"`
	SubCmd       int      `json:"subcmd"`
	HandlerType  string   `json:"handler_type"`
	AuthClientId []string `json:"auth_client_id"`
}

var config = fllag.String("config", "config.json", "configure file path")

func main() {
	flag.Parse()
	data, err := ioutil.ReadFile(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read configure file failed (%s)\n", err)
		return
	}

	cfg := &Config{}
	if err = json.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "parse configure file failed (%s)\n", err)
		return
	}

	xl, err := xlog.New(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create log failed (%s)\n", err)
		return
	}
	defer xl.Close()

	var tlsConfig *tls.Config
	if cfg.TLS != (TLSConfig{}) {
		if cert, err := tls.LoadX509KeyPair(cfg.TLS.Certificate, cfg.TLS.PrivateKey); err == nil {
			tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		} else {
			fmt.Fprintf(os.Stderr, "load tls configure failed (%s)\n", err)
			return
		}
	}

	router := &xp.Router{Logger: xl}
	for _, handler := range cfg.Handlers {
		if err = router.Bind(
			uint32(handler.Cmd),
			uint32(handler.SubCmd),
			GenerateHandler(handler.HandlerType),
			GenerateAuthHandler(handler.AUthClientId)); err != nil {
			fmt.Fprintf(os.Stderr, "bind (cmd: %d subcmd:%d )handler failed (%s)\n", handler.Cmd, handler.SubCmd, err)
			return
		}
	}

	s := &xp.Server{
		Addr:      "0.0.0.0:" + cfg.Port,
		Handler:   router,
		Logger:    xl,
		TLSConfig: tlsConfig,
	}

	if err = xserver.Serve(s); err != nil {
		xl.Fatal("run server failed (%s)", err)
	}
}

func GenerateHandler(handlerType string) xp.Handler {
}

func GenerateAuthHandler(clientId []string) xp.AuthHandler {
}
