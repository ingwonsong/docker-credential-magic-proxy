package main

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/ingwonsong/docker-credential-magic-proxy/pkg/proxy"
)

var (
	proxyPort = flag.Int("proxy-port", 5000, "listening port for the egress proxy")
	allowHTTP = flag.Bool("allow-http-upstream", false, "allow to use HTTP to connect to the upstream registries")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := &http.Server{
			Addr:        ":" + strconv.Itoa(*proxyPort),
			Handler:     proxy.NewHandler(*allowHTTP),
			BaseContext: func(_ net.Listener) context.Context { return ctx },
		}
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("%v", err)
		}
	}()

	// Channel used with signal.Notify should be buffered
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	<-sigChan
	cancel()
	os.Exit(0)
}
