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

		receivedMsg := Deserialize(buf)
		fmt.Printf("Received %d bytes from %s: %v\n", size, source, receivedMsg)
		rcode := 0
		if receivedMsg.Header.Flags.OPCODE != 0 {
			rcode = 4
		}
		msg := DNSMessage{
			Header: DNSHeader{
				ID: receivedMsg.Header.ID,
				Flags: DNSHeaderFlags{
					QR: true,
					OPCODE: receivedMsg.Header.Flags.OPCODE,
					AA: false,
					TC: false,
					RD: receivedMsg.Header.Flags.RD,
					RA: false,
					Z: 0,
					RCODE: uint8(rcode),
				},
				QDCOUNT: 1,
				ANCOUNT: 1,
				NSCOUNT: 0,
				ARCOUNT: 0,
			},
			Question: DNSQuestion{
				Name:  "codecrafters.io",
				Type:  1,
				Class: 1,
			},
			Answer: DNSAnswer{
				Name:  "codecrafters.io",
				Type:  1,
				Class: 1,
				TTL: 60,
				RDLENGTH: 4,
				RDATA: 0,
			},
		}
		fmt.Printf("Sending Message: %v\n", msg)
		// Create a response
		response := msg.Encode()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

func Deserialize(data []byte) DNSMessage {
	return DNSMessage{
		Header: deserializeHeader(data[:12]),
		Question: deserializeQuestion(data[12:]),
	}
}

func deserializeHeader(data []byte) DNSHeader {
	return DNSHeader{
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
	}
}

func deserializeQuestion(data []byte) DNSQuestion {
	counter := 0
	var name string
	for data[counter] != 0x00 {
		partLength := int(data[counter])
		part := make([]byte, partLength)
		copy(part, data[counter+1:counter+1+partLength])
		if name != "" {
			name += "."
		}
		name += string(part)
		counter += partLength + 1
	}

	return DNSQuestion{
		Name: name,
		Type: binary.BigEndian.Uint16(data[1+counter:3+counter]),
		Class: binary.BigEndian.Uint16(data[3+counter:5+counter]),
	}
}
