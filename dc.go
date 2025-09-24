package teowebtransport_server

import (
	"fmt"
	"log"

	"github.com/teonet-go/teowebtransport_server/message"
)

type Dc struct {
	user any
	send func(data []byte) error
}

func (dc *Dc) GetUser() any           { return dc.user }
func (dc *Dc) SetUser(user any)       { dc.user = user }
func (dc *Dc) Send(data []byte) error { return dc.send(data) }

func (*Webtransport) newDc(user any, send func(data []byte) error) *Dc {
	return &Dc{user: user, send: send}
}

// GetDc returns DataChannel for selected login.
//
// If the login not found the function creates new DataChannel and sets it to
// the map.
func (w *Webtransport) GetDc(login string, sendAnswer func([]byte) error) (dc *Dc) {

	// Get DataChannel
	dc, ok := w.dc.Get(login)
	if ok {
		return
	}

	// Create new DataChannel
	dc = w.newDc(login, sendAnswer)
	w.dc.Set(login, dc)
	w.sendClientsToAll()
	log.Printf("created new dc for login: %s\n", login)

	// Execute subscribe/clients command subscription
	if w.sub != nil {
		w.sub.ExecCmd("clients")
	}

	return
}

// GetClients returns number of clients.
func (w *Webtransport) getClients() int {
	return w.dc.Len()
}

func (w *Webtransport) sendClientsToAll() {
	data := fmt.Appendf(nil, "%d", w.getClients())

	outMsg := &message.Message{
		Command: "clients",
		Data:    data,
	}
	data, _ = outMsg.MarshalBinary()

	for _, dc := range w.dc.Range {
		dc.Send(data)
	}
}
