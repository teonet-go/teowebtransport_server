// Copyright 2025 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Webtransport teonet server package.
// This package create WebTransport server to connect teonet web clients and
// process it commands.
package teowebtransport_server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kirill-scherba/command/v2"
	"github.com/teonet-go/teowebtransport_server/message"
	"github.com/teonet-go/webtransport-go"
)

// Webtransport server struct
type Webtransport struct {
	// WebTransport server connection
	server *webtransport.Server

	// Available teonet commands
	*command.Commands

	// Connected peers
	// peers
}

// Webtransport server config
type Config struct {
	// ListenAddr sets an address to bind server to, e.g. ":4433"
	ListenAddr string

	// TLSCert defines a path or data to TLS certificate (CRT file)
	TLSCert webtransport.CertFile

	// TLSKey defines a path or data to TLS certificate's private key (KEY file)
	TLSKey webtransport.CertFile

	// AllowedOrigins represents list of allowed origins to connect from
	AllowedOrigins []string

	// KeepAlivePeriod defines whether this peer will periodically send a packet
	// to keep the connection alive. If set to 0, then no keep alive is sent.
	// Otherwise, the keep alive is sent on that period (or at most every half
	// of MaxIdleTimeout, whichever is smaller).
	KeepAlivePeriod time.Duration

	// MaxIdleTimeout is the maximum duration that may pass without any incoming
	// network activity. The actual value for the idle timeout is the minimum of
	// this value and the peer's. This value only applies after the handshake
	// has completed.
	//
	// If the timeout is exceeded, the connection is closed.
	// If this value is zero, the timeout is set to 30 seconds.
	MaxIdleTimeout time.Duration
}

// Create new teonet webtransport server.
func New(conf *Config, commands *command.Commands) (t *Webtransport) {

	t = &Webtransport{Commands: commands}

	// Create a WebTransport server
	t.server = &webtransport.Server{
		ListenAddr:     conf.ListenAddr,
		TLSCert:        conf.TLSCert,
		TLSKey:         conf.TLSKey,
		AllowedOrigins: conf.AllowedOrigins,
		QuicConfig: &webtransport.QuicConfig{
			KeepAlivePeriod: conf.KeepAlivePeriod,
			MaxIdleTimeout:  conf.MaxIdleTimeout,
		},
	}

	return
}

// Run server
func (t *Webtransport) Run(ctx context.Context) error {

	// Register HTTP/3 connection handler
	http.HandleFunc("/wt", func(rw http.ResponseWriter, r *http.Request) {

		// Get and accept incoming WebTransport session
		session := r.Body.(*webtransport.Session)
		session.AcceptSession()
		// session.RejectSession(400)
		log.Println("accepted incoming webtransport session")

		// Handle incoming webtransport streams
		t.handleStreams(session)

		// Open outgoing server-initiated bidirectional webtransport streams
		s, err := session.OpenStreamSync(session.Context())
		// s, err := session.OpenStreamSync(ctx)
		if err != nil {
			log.Println(err)
		}
		log.Printf("listening on server-initiated bidi stream %v\n", s.StreamID())

		// Send a message to server-initiated bidi stream
		sendMsg := []byte("bidi")
		log.Printf("sending to server-initiated bidi stream %v: %s\n", s.StreamID(), sendMsg)
		s.Write(sendMsg)

		// Process messages from server-initiated bidi stream
		go func(s webtransport.Stream) {
			defer s.Close()
			for {
				buf := make([]byte, 1024)
				n, err := s.Read(buf)
				if err != nil {
					log.Printf("error reading from server-initiated bidi stream %v: %v\n", s.StreamID(), err)
					break
				}
				log.Printf("received from server-initiated bidi stream %v: %s\n", s.StreamID(), buf[:n])
			}
		}(s)

		// Open outgoing server-initiated unidirectional webtransport stream
		sUni, err := session.OpenUniStreamSync(session.Context())
		// sUni, err := session.OpenUniStreamSync(ctx)
		if err != nil {
			log.Println(err)
		}

		// Send a message to server-initiated uni stream
		sendMsg = []byte("uni")
		log.Printf("sending to server-initiated uni stream %v: %s\n", s.StreamID(), sendMsg)
		sUni.Write(sendMsg)
	})

	return t.server.Run(ctx)
}

// handleStreams handles incoming webtransport streams.
func (t *Webtransport) handleStreams(session *webtransport.Session) {

	// Handle incoming datagrams
	go func() {
		for {
			msg, err := session.ReceiveDatagram(session.Context())
			if err != nil {
				log.Println("session closed, ending datagram listener:", err)
				break
			}
			log.Printf("received datagram: %s\n", msg)

			sendMsg := bytes.ToUpper(msg)
			log.Printf("sending datagram: %s\n", sendMsg)
			session.SendDatagram(sendMsg)
		}
	}()

	// Handle incoming unidirectional streams.
	go func() {
		for {
			s, err := session.AcceptUniStream(session.Context())
			if err != nil {
				log.Println("session closed, not accepting more uni streams:", err)
				break
			}
			log.Println("accepting incoming uni stream:", s.StreamID())

			go func(s webtransport.ReceiveStream) {
				for {
					buf := make([]byte, 1024)
					n, err := s.Read(buf)
					if err != nil {
						log.Printf("error reading from uni stream %v: %v\n", s.StreamID(), err)
						break
					}
					log.Printf("received from uni stream: %s\n", buf[:n])
				}
			}(s)
		}
	}()

	// Handle incoming bidirectional streams and process messages from them.
	go func() {
		for {
			s, err := session.AcceptStream()
			if err != nil {
				log.Println("session closed, not accepting more bidi streams:", err)
				break
			}
			log.Println("accepting incoming bidi stream:", s.StreamID())

			go func(s webtransport.Stream) {
				reader := message.NewMessageReader(s)
				defer s.Close()
				for {
					// Read and unmarshal message
					msg, err := reader.Read()
					if err != nil {
						log.Printf("error reading from bidi stream %v: %v\n",
							s.StreamID(), err)
						break
					}
					log.Printf(
						"received from bidi stream %v, cmd: %s, data len: %d\n",
						s.StreamID(), msg.Command, len(msg.Data))

					// Process message and send answer
					outMsg, _ := t.processMessage(msg)
					data, _ := outMsg.MarshalBinary()
					s.Write(data)
				}
			}(s)
		}
	}()
}

func (t *Webtransport) processMessage(in *message.Message) (
	out *message.Message, err error) {

	// Make default output message containing input message fields
	out = &message.Message{
		ID:      in.ID,
		Address: in.Address,
		Command: in.Command,
	}

	// Print message command to console
	log.Println("received command:", string(in.Command))

	// Parse message
	_, name, vars, _, _ := t.Commands.ParseCommand([]byte(in.Command))

	// Execute command
	log.Println("executing command:", name, vars)
	reader, err := t.Commands.Exec(name, command.WebRTC,
		&WebtransportRequest{Vars: vars, Data: in.Data},
	)
	if err != nil {
		err = fmt.Errorf("failed to execute: %w", err)
		out.Data = []byte(err.Error())
		log.Println(err)
		out.Err = 1
		return
	}

	// Get data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		err = fmt.Errorf("failed to read answer data: %w", err)
		out.Data = []byte(err.Error())
		log.Println(err)
		out.Err = 1
		return
	}

	// Set data to output message
	out.Data = data

	return
}
