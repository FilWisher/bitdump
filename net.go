package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

const (
	FIN = 1 << 0
	SYN = 1 << 1
	RST = 1 << 2
	PSH = 1 << 3
	ACK = 1 << 4
	URG = 1 << 5
)

type TCPHeader struct {
	Source      uint16
	Destination uint16
	SeqNum      uint32
	AckNum      uint32
	DataOffset  uint8 // 4 bits
	Reserved    uint8 // 3 bits
	ECN         uint8 // 3 bits
	Ctrl        uint8 // 6 bits
	Window      uint16
	Checksum    uint16 // Kernel will set this if it's 0
	Urgent      uint16
	Options     []TCPOption
}

type TCPOption struct {
	Kind   uint8
	Length uint8
	Data   []byte
}

// Parse packet into TCPHeader structure
func NewTCPHeader(data []byte) (*TCPHeader, error) {
	var tcp TCPHeader
	var err error

	r := bytes.NewReader(data)
	err = binary.Read(r, binary.BigEndian, &tcp.Source)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &tcp.Destination)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &tcp.SeqNum)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &tcp.AckNum)
	if err != nil {
		return nil, err
	}

	var mix uint16
	binary.Read(r, binary.BigEndian, &mix)
	tcp.DataOffset = byte(mix >> 12)  // top 4 bits
	tcp.Reserved = byte(mix >> 9 & 7) // 3 bits
	tcp.ECN = byte(mix >> 6 & 7)      // 3 bits
	tcp.Ctrl = byte(mix & 0x3f)       // bottom 6 bits

	err = binary.Read(r, binary.BigEndian, &tcp.Window)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &tcp.Checksum)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &tcp.Urgent)
	if err != nil {
		return nil, err
	}

	return &tcp, nil
}

func main() {
	netaddr, err := net.ResolveIPAddr("ip4", "0.0.0.0")
	if err != nil {
		log.Println(err)
		return
	}
	conn, err := net.ListenIP("ip4:tcp", netaddr)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		buf := make([]byte, 4096)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Println(err)
			continue
		}

		tcphdr, err := NewTCPHeader(buf[:n])
		if err != nil {
			log.Println(err)
			continue
		}

		// No TCP payload
		length := n - (int(tcphdr.DataOffset) * 4)
		if length <= 0 {
			continue
		}

		// Offset larger than number of bytes read
		if int(tcphdr.DataOffset)*4 >= n {
			continue
		}

		data := buf[int(tcphdr.DataOffset)*4 : n]

		hs, err := NewHandshake(data)
		if err == nil {
			fmt.Printf("%v\n", hs)
			continue
		}

		msg, err := NewMessage(data)
		if err != nil {
			continue
		}
		fmt.Printf("%v\n", msg)
	}
}
