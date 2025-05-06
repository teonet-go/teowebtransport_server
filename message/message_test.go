package message

import (
	"testing"
)

func TestMessage(t *testing.T) {

	// Marshal a message
	msg := &Message{
		ID:      11,
		Command: "get",
		Data:    []byte("data"),
	}
	data, _ := msg.MarshalBinary()
	t.Log(data)

	// Unmarshal
	msg2 := &Message{}
	_ = msg2.UnmarshalBinary(data)

	// Compare messages
	if msg.ID != msg2.ID {
		t.Errorf("ID: %d != %d", msg.ID, msg2.ID)
	}
	if msg.Address != msg2.Address {
		t.Errorf("Address: %s != %s", msg.Address, msg2.Address)
	}
	if msg.Command != msg2.Command {
		t.Errorf("Command: %s != %s", msg.Command, msg2.Command)
	}
	if string(msg.Data) != string(msg2.Data) {
		t.Errorf("Data: %s != %s", string(msg.Data), string(msg2.Data))
	}

}