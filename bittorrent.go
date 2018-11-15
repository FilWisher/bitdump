package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type MessageType uint8

const (
	Choke MessageType = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
	Port
)

func (m MessageType) String() string {
	switch m {
	case Choke:
		return "Choke"
	case Unchoke:
		return "Unchoke"
	case Interested:
		return "Interested"
	case NotInterested:
		return "NotInterested"
	case Have:
		return "Have"
	case Bitfield:
		return "Bitfield"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	case Cancel:
		return "Cancel"
	case Port:
		return "Port"
	}
	return "Unknown"
}

type Message struct {
	Len    uint32
	Type   MessageType
	Index  uint32
	Begin  uint32
	Length uint32
	Bytes  []byte
	Port   uint16
}

const Protocol = "BitTorrent protocol"

type Handshake struct {
	Protocol string
	InfoHash [20]byte
	PeerID   [20]byte
}

func (hs Handshake) String() string {
	return fmt.Sprintf("Handshake: %s: %X %X", hs.Protocol, hs.InfoHash, hs.PeerID)
}

func NewHandshake(buf []byte) (*Handshake, error) {
	var hs Handshake

	r := bytes.NewReader(buf)
	var length uint8
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	protocol := make([]byte, length)
	err = binary.Read(r, binary.BigEndian, &protocol)
	if err != nil {
		return nil, err
	}

	hs.Protocol = string(protocol)
	if hs.Protocol != "BitTorrent protocol" {
		return nil, fmt.Errorf("Not a valid protocol")
	}

	var reserved uint64
	err = binary.Read(r, binary.BigEndian, &reserved)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &hs.InfoHash)
	if err != nil {
		fmt.Println("HERE")
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &hs.InfoHash)
	if err != nil {
		return nil, err
	}

	return &hs, nil
}

type ErrorType uint8

const (
	IncorrectLength ErrorType = iota
	UnrecognizedMessage
)

type BittorrentError struct {
	Type ErrorType
	Msg  string
}

func (e BittorrentError) Error() string {
	return e.Msg
}

func NewIncorrectLength(m Message) error {
	return BittorrentError{
		IncorrectLength,
		fmt.Sprintf("Incorrect message length for %s: %d", m.Type, m.Len),
	}
}

func NewUnrecognizedMessage(m Message) error {
	return BittorrentError{
		IncorrectLength,
		fmt.Sprintf("Unrecognized message type"),
	}
}

func NewMessage(buf []byte) (*Message, error) {
	var msg Message

	r := bytes.NewReader(buf)

	err := binary.Read(r, binary.BigEndian, &msg.Len)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &msg.Type)
	if err != nil {
		return nil, err
	}

	switch msg.Type {
	case Choke:
		fallthrough
	case Unchoke:
		fallthrough
	case Interested:
		fallthrough
	case NotInterested:
		if msg.Len != 1 {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Have:
		if msg.Len != 5 {
			return nil, NewIncorrectLength(msg)
		}
		err = binary.Read(r, binary.BigEndian, &msg.Index)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	case Request:
		if msg.Len != 13 {
			return nil, NewIncorrectLength(msg)
		}
		err = binary.Read(r, binary.BigEndian, &msg.Index)
		if err != nil {
			return nil, err
		}
		err = binary.Read(r, binary.BigEndian, &msg.Begin)
		if err != nil {
			return nil, err
		}
		err = binary.Read(r, binary.BigEndian, &msg.Length)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	case Cancel:
		if msg.Len != 13 {
			return nil, NewIncorrectLength(msg)
		}
		err = binary.Read(r, binary.BigEndian, &msg.Index)
		if err != nil {
			return nil, err
		}
		err = binary.Read(r, binary.BigEndian, &msg.Begin)
		if err != nil {
			return nil, err
		}
		err = binary.Read(r, binary.BigEndian, &msg.Length)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	case Bitfield:
		if msg.Len < 1 || msg.Len > uint32(len(buf)) {
			return nil, NewIncorrectLength(msg)
		}
		msg.Bytes = make([]byte, msg.Len-1)
		err = binary.Read(r, binary.BigEndian, &msg.Bytes)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	case Piece:
		if msg.Len < 9 || msg.Len > uint32(len(buf)) {
			return nil, NewIncorrectLength(msg)
		}
		err = binary.Read(r, binary.BigEndian, &msg.Index)
		if err != nil {
			return nil, err
		}
		err = binary.Read(r, binary.BigEndian, &msg.Begin)
		if err != nil {
			return nil, err
		}
		msg.Bytes = make([]byte, msg.Len-9)
		err = binary.Read(r, binary.BigEndian, &msg.Bytes)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	case Port:
		if msg.Len != 3 {
			return nil, NewIncorrectLength(msg)
		}
		err = binary.Read(r, binary.BigEndian, &msg.Port)
		if err != nil {
			return nil, err
		}
		return &msg, nil
	default:
		return nil, NewUnrecognizedMessage(msg)
	}

	return &msg, nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (m *Message) String() string {
	switch m.Type {
	case Have:
		return fmt.Sprintf("%s %d", m.Type, m.Index)
	case Request:
		return fmt.Sprintf("%s %d %d %d", m.Type, m.Index, m.Begin, m.Length)
	case Cancel:
		return fmt.Sprintf("%s %d %d %d", m.Type, m.Index, m.Begin, m.Length)
	case Bitfield:
		upper := min(len(m.Bytes), 20)
		return fmt.Sprintf("%s %x ", m.Type, m.Bytes[:upper])
	case Piece:
		return fmt.Sprintf("%s %d %d `...`", m.Type, m.Index, m.Begin)
	case Port:
		return fmt.Sprintf("%s %d `...`", m.Type, m.Port)
	default:
		return m.Type.String()
	}
}
