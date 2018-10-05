// Package mcgorcon is a Minecraft RCON Client written in Go.
// It is designed to be easy to use and integrate into your own applications.
package mcgorcon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"errors"
	"io"
	"net"
	"time"
)

type packetType int32

// Client is a representation of an RCON client.
type Client struct {
	password   string
	connection net.Conn
}

// header is the header of a Minecraft RCON packet.
type header struct {
	Size       int32
	RequestID  int32
	PacketType packetType
}

const packetTypeCommand packetType = 2
const packetTypeAuth packetType = 3
const requestIDBadLogin int32 = -1

// Dial up the server and establish a RCON conneciton.
func Dial(host string, port int, pass string) (Client, error) {
	// Combine the host and port to form the address.
	address := host + ":" + fmt.Sprint(port)
	// Actually establish the conneciton.
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return Client{}, err
	}
	// Create the client object, since the connection has been established.
	c := Client{password: pass, connection: conn}
	// TODO - server validation to make sure we're talking to a real RCON server.
	// For now, just return the client and assume it's a real server.
	return c, nil
}

// SendCommand sends a command to the server and returns the result (often nothing).
func (c *Client) SendCommand(command string) (string, error) {
	// Because I'm lazy, just authenticate with every command.
	err := c.authenticate()
	if err != nil {
		return "", err
	}

	// Send the packet.
	head, payload, err := c.sendPacket(packetTypeCommand, []byte(command))
	if err != nil {
		return "", err
	}

	// Auth was bad, throw error.
	if head.RequestID == requestIDBadLogin {
		return "", errors.New("Bad auth, could not send command.")
	}
	return string(payload), nil
}

// authenticate authenticates the user with the server.
func (c *Client) authenticate() error {
	// Send the packet.
	head, _, err := c.sendPacket(packetTypeAuth, []byte(c.password))
	if err != nil {
		return err
	}

	// If the credentials were bad, throw error.
	if head.RequestID == requestIDBadLogin {
		return errors.New("Bad auth, could not authenticate.")
	}

	return nil
}

// sendPacket sends the binary packet representation to the server and returns the response.
func (c *Client) sendPacket(t packetType, p []byte) (header, []byte, error) {
	// Generate the binary packet.
	packet, err := packetise(t, p)
	if err != nil {
		return header{}, nil, err
	}

	// Send the packet over the wire.
	_, err = c.connection.Write(packet)
	if err != nil {
		return header{}, nil, err
	}
	// Receive and decode the response.
	head, payload, err := depacketise(c.connection)
	if err != nil {
		return header{}, nil, err
	}

	return head, payload, nil
}

// packetise encodes the packet type and payload into a binary representation to send over the wire.
func packetise(t packetType, p []byte) ([]byte, error) {
	// Generate a random request ID.
	pad := [2]byte{}
	length := int32(len(p) + 10)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, length)
	binary.Write(&buf, binary.LittleEndian, int32(0))
	binary.Write(&buf, binary.LittleEndian, t)
	binary.Write(&buf, binary.LittleEndian, p)
	binary.Write(&buf, binary.LittleEndian, pad)
	// Notchian server doesn't like big packets :(
	if buf.Len() >= 1460 {
		return nil, errors.New("Packet too big when packetising.")
	}
	// Return the bytes.
	return buf.Bytes(), nil
}

// depacketise decodes the binary packet into a native Go struct.
func depacketise(r io.Reader) (header, []byte, error) {
	head := header{}
	err := binary.Read(r, binary.LittleEndian, &head)
	if err != nil {
		return header{}, nil, err
	}
	payload := make([]byte, head.Size-8)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return header{}, nil, err
	}
	return head, payload[:len(payload)-2], nil
}
