package teowebtransport_server

import (
	"time"

	"github.com/kirill-scherba/command/v2/subscription"
	"github.com/teonet-go/teogw"
)

// WebtransportRequest contains gorilla websocket connection and variables map.
type WebtransportRequest struct {
	Vars map[string]string
	Dc   *Dc
	Gw   *teogw.TeogwData
}

// GetVars returns map of request variables.
func (r *WebtransportRequest) GetVars() map[string]string { return r.Vars }

// GetData returns request data.
func (r *WebtransportRequest) GetData() []byte { return r.Gw.Data }

// GetConnectionChannel returns connection channel.
func (r *WebtransportRequest) GetConnectionChannel() subscription.ConnectionChannel { return r.Dc }

// SetDate sets date to responce. Used in HTTP request and set custom date
// to HTTP writer.
func (r *WebtransportRequest) SetDate(contentType string, date time.Time) {}
