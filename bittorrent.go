package main

import (
	"fmt"
	"bytes"
	"encoding/binary"
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
	Length uint32
	Type MessageType
}

const Protocol = "BitTorrent protocol"

type Handshake struct {
	Protocol string
	InfoHash [20]byte
	PeerID   [20]byte
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
	Msg string
}

func (e BittorrentError) Error() string {
	return e.Msg
}

func NewIncorrectLength(m Message) error {
	return BittorrentError{
		IncorrectLength,
		fmt.Sprintf("Incorrect message length for %s: %d", m.Type, m.Length),
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

	err := binary.Read(r, binary.BigEndian, &msg.Length)
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
		if msg.Length != 1 {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Have:
		if msg.Length != 5 {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Request:
		fallthrough
	case Cancel:
		if msg.Length != 13 {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Bitfield:
		if msg.Length < 1 || msg.Length > uint32(len(buf)) {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Piece:
		if msg.Length < 9 || msg.Length > uint32(len(buf)) {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	case Port:
		if msg.Length != 3 {
			return nil, NewIncorrectLength(msg)
		}
		return &msg, nil
	default:
		return nil, NewUnrecognizedMessage(msg)
	}

	return &msg, nil
}
