package main

import (
	"encoding/binary"
	"fmt"
	// Uncomment this block to pass the first stage
	"net"
)

type DNSMessage struct {
	Header DNSResponseHeader
	content string
}

type DNSHeaderFlags struct {
	QR bool // Query Response
	OPCODE uint8 // Operation Code
	AA bool // Authoritative Answer
	TC bool // Truncate Message
	RD bool // Recursive Desired
	RA bool // Recursion Available
	Z uint8 // Reserved
	RCODE uint8 // Response Code
}

type DNSResponseHeader struct {
	ID uint16 // Packet Identifier
	Flags DNSHeaderFlags
	QDCOUNT uint16 // Question Count
	ANCOUNT uint16 // Answer Count
	NSCOUNT uint16 // Authority Count
	ARCOUNT uint16 // Additional Count
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
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
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		// Create a response
		header := CreateResponse();
		response := Serialize(header)
		fmt.Printf("Serialized response header bytes %b", response)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

func Serialize(header DNSResponseHeader) []byte {
	result := make([]byte, binary.Size(header))
	binary.BigEndian.PutUint16(result[:2], header.ID)

	var flags uint16
	if header.Flags.QR {
		flags |= 1 << 15
	}
	flags |= uint16(header.Flags.OPCODE) << 11
	if header.Flags.AA {
		flags |= 1 << 10
	}
	if header.Flags.TC {
		flags |= 1 << 9
	}
	if header.Flags.RD {
		flags |= 1 << 8
	}
	if header.Flags.RA {
		flags |= 1 << 7
	}
	flags |= uint16(header.Flags.Z) << 4
	flags |= uint16(header.Flags.RCODE)
	binary.BigEndian.PutUint16(result[2:4], flags)

	binary.BigEndian.PutUint16(result[4:6], header.QDCOUNT)
	binary.BigEndian.PutUint16(result[6:8], header.ANCOUNT)
	binary.BigEndian.PutUint16(result[8:10], header.NSCOUNT)
	binary.BigEndian.PutUint16(result[10:12], header.ARCOUNT)

	return result
}

func CreateResponse() DNSResponseHeader {
	return DNSResponseHeader{
		ID: 1234,
		Flags: DNSHeaderFlags {
			QR: true,
			OPCODE: 0x0,
			AA: false,
			TC: false,
			RD: false,
			RA: false,
			Z: 0x0,
			RCODE: 0x0,
		},
		QDCOUNT: 0x0,
		ANCOUNT: 0x0,
		NSCOUNT: 0x0,
		ARCOUNT: 0x0,
	}
}
