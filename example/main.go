package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kirill-scherba/command/v2"
	server "github.com/teonet-go/teowebtransport_server"
	"github.com/teonet-go/webtransport-go"
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

	// Generate local TLS certificate
	// cert, key, err := generateCert()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Create teonet webtransport server
	server := server.New(&server.Config{
		ListenAddr: addr,
		TLSCert:    webtransport.CertFile{Path: "asuzs.teonet.dev.crt"},
		TLSKey:     webtransport.CertFile{Path: "asuzs.teonet.dev.key"},
		// TLSCert: webtransport.CertFile{Path: "localhost.crt"},
		// TLSKey:  webtransport.CertFile{Path: "localhost.key"},
		// TLSCert: webtransport.CertFile{Data: cert},
		// TLSKey:  webtransport.CertFile{Data: key},
		AllowedOrigins: []string{
			"googlechrome.github.io",
			// "127.0.0.1:8099",
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

// generateCert generates a new TLS certificate and key pair using the given
// parameters. It returns the certificate and key in PEM format.
func generateCert() (certPEM []byte, keyPEM []byte, err error) {
	// Generate a new RSA key
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return
	}

	// Create a new certificate
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return
	}

	// Encode the key and certificate in PEM format
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return
}

// func loadTLSConfig() (tlsConfig *tls.Config) {

// 	var err error

// 	tlsConfig = &tls.Config{}

// 	// Add a certificate and key
// 	tlsConfig.Certificates = make([]tls.Certificate, 1)
// 	tlsConfig.Certificates[0], err = tls.LoadX509KeyPair("asuzs.teonet.dev.crt", "asuzs.teonet.dev.key")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	tlsConfig.NextProtos = []string{"quic"}

// 	return
// }

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
