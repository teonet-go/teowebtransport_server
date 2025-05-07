// Package message provides a way to read messages from a stream.
//
// A message is a sequence of bytes that is prefixed with a 4-byte big-endian
// unsigned integer that specifies the length of the message.
//
// The StreamReader type provides a way to read messages from a stream. It
// provides a ReadMessage method that reads a single message from the stream and
// returns it as a Message value. It also provides a ReadMessages method that
// reads multiple messages from the stream and returns them as a slice of Message
// values.
//
// The Message type represents a single message. It contains a Data field that
// contains the message data.
//
// The ReadMessage method reads a single message from the stream. It first reads
// the message length from the stream and then reads the message data from the
// stream. It returns a Message value that contains the message data.
//
// The ReadMessages method reads multiple messages from the stream. It reads
// messages from the stream until it reaches the end of the stream. It returns a
// slice of Message values that contains the messages.

package message

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Message represents a single message. It matches WebRTCData interface.
type Message struct {
	ID      uint32
	Address string
	Command string
	Data    []byte
	Err     byte
	length  uint32
}

// WebRTCData interface methods
func (m *Message) GetID() uint32      { return m.ID }
func (m *Message) GetAddress() string { return m.Address }
func (m *Message) GetCommand() string { return m.Command }
func (m *Message) GetData() []byte    { return m.Data }

// serviceFieldsLength consists of the id field length (4 bytes), 
// the length of the date length field (4 bytes),
// and the error byte length (1 byte)
const serviceFieldsLength = 4 + 4 + 1

// MarshalBinary encodes a message as a byte array.
func (m *Message) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// Message binary format
	//--------------------------------------------------------------
	// Length  | ID      | Data length | Command | Data    | Error
	//--------------------------------------------------------------
	// 4 bytes | 4 bytes | 4 bytes     | n bytes | n bytes | 1 byte
	//--------------------------------------------------------------

	// Message length consists of the id length, the command length,
	// the data length, and the size of the variable containing the data length
	// and error byte
	length := uint32(len(m.Command) + len(m.Data) + serviceFieldsLength)
	if err := binary.Write(&buf, binary.BigEndian, length); err != nil {
		return nil, err
	}

	// Message id
	if err := binary.Write(&buf, binary.BigEndian, m.ID); err != nil {
		return nil, err
	}

	// Data length
	dataLength := uint32(len(m.Data))
	if err := binary.Write(&buf, binary.BigEndian, dataLength); err != nil {
		return nil, err
	}

	// Message command
	if _, err := buf.WriteString(m.Command); err != nil {
		return nil, err
	}

	// Message data
	if dataLength > 0 {
		buf.Write(m.Data)
	}

	// Message error
	if err := binary.Write(&buf, binary.BigEndian, m.Err); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary decodes a message from a byte array.
func (m *Message) UnmarshalBinary(data []byte) (err error) {

	reader := bytes.NewReader(data)

	// Get length of message
	if m.length == 0 {
		err = binary.Read(reader, binary.BigEndian, &m.length)
		if err != nil {
			return err
		}
	}

	// Get id of message
	err = binary.Read(reader, binary.BigEndian, &m.ID)
	if err != nil {
		return
	}

	// Get data length of message
	var dataLength uint32
	err = binary.Read(reader, binary.BigEndian, &dataLength)
	if err != nil {
		return
	}

	// Get command of message
	command := make([]byte, m.length-dataLength-serviceFieldsLength)
	_, err = io.ReadFull(reader, command)
	if err != nil {
		return
	}
	m.Command = string(command)

	// Get data of message
	if dataLength > 0 {
		m.Data = make([]byte, dataLength)
		_, err = io.ReadFull(reader, m.Data)
		if err != nil {
			return
		}
	}

	// Get error of message
	err = binary.Read(reader, binary.BigEndian, &m.Err)
	if err != nil {
		return
	}

	return
}

// MessageReader reads data from a stream and splits it into messages.
type MessageReader struct {
	stream io.Reader
}

// NewMessageReader creates and returns a new StreamReader.
func NewMessageReader(stream io.Reader) *MessageReader {
	return &MessageReader{stream: stream}
}

// Read reads and decodes single message from the stream.
func (r *MessageReader) Read() (message *Message, err error) {
	// Read the message length from the stream
	var length uint32
	err = binary.Read(r.stream, binary.BigEndian, &length)
	if err != nil {
		return
	}

	// Read the message data from the stream
	data := make([]byte, length)
	_, err = io.ReadFull(r.stream, data)
	if err != nil {
		return
	}

	// Unmarshal the message data
	message = &Message{length: length}
	err = message.UnmarshalBinary(data)
	if err != nil {
		return
	}

	return
}
