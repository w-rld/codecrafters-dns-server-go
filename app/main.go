package main

import (
	"encoding/binary"
	"fmt"
	"net"
)


func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])

		receivedMsg := Deserialize(buf)
		fmt.Printf("Received %d bytes from %s: %v\n", size, source, receivedMsg)
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)
		msg := DNSMessage{
			Header: DNSHeader{
				ID: 1234,
				Flags: DNSHeaderFlags{
					QR: true,
				},
				QDCOUNT: 1,
			},
			Question: DNSQuestion{
				Name:  "codecrafters.io",
				Type:  1,
				Class: 1,
			},
		}
		fmt.Printf("Sending Message: %v\n", msg)
		// Create a response
		response := msg.Encode()

		fmt.Printf("Serialized response bytes %b\n", response)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

func Deserialize(data []byte) DNSMessage {
	return DNSMessage{Header:DNSHeader{
		ID: binary.BigEndian.Uint16(data[:2]),
		Flags: DNSHeaderFlags{
			QR: (data[2] & 0x80) != 0,
			OPCODE: (data[2] >> 3) & 0x0F,
			AA: (data[2] & 0x04) != 0,
			TC: (data[2] & 0x02) != 0,
			RD: (data[2] & 0x01) != 0,
			RA: (data[2] & 0x80) != 0,
			Z: (data[3] >> 4) & 0x07,
			RCODE: data[3] & 0x0F,
			},
			QDCOUNT: binary.BigEndian.Uint16(data[4:6]),
			ANCOUNT: binary.BigEndian.Uint16(data[6:8]),
			NSCOUNT: binary.BigEndian.Uint16(data[8:10]),
			ARCOUNT: binary.BigEndian.Uint16(data[10:12]),
			},
		Question: DNSQuestion{
			Name: string(data[12:14]),
			Type: binary.BigEndian.Uint16(data[14:16]),
			Class: binary.BigEndian.Uint16(data[16:18]),
		},
	}
}
