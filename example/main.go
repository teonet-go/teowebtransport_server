package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kirill-scherba/command/v2"
	server "github.com/teonet-go/teowebtransport_server"
)

func main() {

	// Set log level with microseconds
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	const listenAddrHTTP = ":8099"
	const listenAddrHTTP3 = ":4433"

	// Start http server
	go serve(listenAddrHTTP)

	// Start http/3 server
	serve3(listenAddrHTTP3, commands())
}

// serve define HTTP handlers and start http server to serve static files.
func serve(addr string) {
	// Static part of frontend
	frontendFS := http.FileServer(http.FS(os.DirFS("./")))
	http.Handle("/", frontendFS)

	log.Println("http server is listening on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// serve3 define HTTP/3 handlers and start http/3 server to provide webtransport.
func serve3(addr string, commands *command.Commands) {
	// Create teonet webtransport server
	server := server.New(&server.Config{
		ListenAddr: addr,
		TLSCert:    "asuzs.teonet.dev.crt",
		TLSKey:     "asuzs.teonet.dev.key",
		AllowedOrigins: []string{
			"googlechrome.github.io",
			"127.0.0.1:8099",
			"localhost:8099",
			"localhost:8082",
			"new-tab-page",
			"",
		},
		KeepAlivePeriod: 30 * time.Second,
		MaxIdleTimeout:  30 * time.Second,
	}, commands)

	// Run teonet webtransport server
	log.Println("http/3 server is listening on", addr)
	ctx := context.Background()
	err := server.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

// commans creates commands for http/3 server and returns them.
func commands() (c *command.Commands) {
	c = command.New()

	c.Add("hello", "Test command",
		command.WebTransport, "{name}",
		"", "", "",
		func(cmd *command.CommandData, processIn command.ProcessIn, data any) (
			out io.Reader, err error) {

			// Get input vars from data
			vars, err := c.Vars(data)
			if err != nil {
				return
			}

			out = strings.NewReader(fmt.Sprintf("Hello %s!", vars["name"]))
			return
		},
	)

	return
}
