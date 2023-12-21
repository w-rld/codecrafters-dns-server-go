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
					OPCODE: 0,
					AA: false,
					TC: false,
					RD: false,
					RA: false,
					Z: 0,
					RCODE: 0,
				},
				QDCOUNT: 1,
				ANCOUNT: 0,
				NSCOUNT: 0,
				ARCOUNT: 0,
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
		test := Deserialize(response)
		test2 := deserializeQuestion(msg.Question.Encode())
		fmt.Printf("Deserialized Serialized response bytes %v\n", test)
		fmt.Printf("Deserialized Serialized question bytes %v\n", test2)

		fmt.Printf("Serialized response bytes %b\n", response)

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
	questionNameLength := int(data[0])
	questionName := make([]byte, questionNameLength)
	copy(questionName, data[1:1+questionNameLength])
	return DNSQuestion{
		Name: string(questionName),
		Type: binary.BigEndian.Uint16(data[1+questionNameLength:3+questionNameLength]),
		Class: binary.BigEndian.Uint16(data[3+questionNameLength:5+questionNameLength]),
	}
}
