package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
)

func main() {
	resolver := flag.String("resolver", "", "resolver address")
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

		if source.IP.String() == *resolver {

		}
		fmt.Printf("Received %d bytes from %s: %v\n", size, source, receivedMsg)
		var response []byte
		rcode := 0
		if receivedMsg.Header.Flags.OPCODE != 0 {
			rcode = 4
		}
		resMsg := DNSMessage{
			Header: DNSHeader{
				ID: receivedMsg.Header.ID,
				Flags: DNSHeaderFlags{
					QR:     true,
					OPCODE: receivedMsg.Header.Flags.OPCODE,
					AA:     false,
					TC:     false,
					RD:     receivedMsg.Header.Flags.RD,
					RA:     false,
					Z:      0,
					RCODE:  uint8(rcode),
				},
				QDCOUNT: 0,
				ANCOUNT: 0,
				NSCOUNT: 0,
				ARCOUNT: 0,
			},
			Questions: []DNSQuestion{},
			Answers:   []DNSAnswer{},
		}
		if *resolver != "" {
			addr, err := net.ResolveUDPAddr("udp", *resolver)
			if err != nil {
				fmt.Println("Failed to resolve UDP address:", err)
				return
			}

			udp, err := net.ListenUDP("udp", addr)
			if err != nil {
				fmt.Println("Failed to bind to address:", err)
				return
			}
			for _, question := range receivedMsg.Questions {
				msg := DNSMessage{
					Header: DNSHeader{
						ID: receivedMsg.Header.ID,
						Flags: DNSHeaderFlags{
							QR:     true,
							OPCODE: receivedMsg.Header.Flags.OPCODE,
							AA:     false,
							TC:     false,
							RD:     receivedMsg.Header.Flags.RD,
							RA:     false,
							Z:      0,
							RCODE:  uint8(rcode),
						},
						QDCOUNT: 1,
						ANCOUNT: 0,
						NSCOUNT: 0,
						ARCOUNT: 0,
					},
					Questions: []DNSQuestion{question},
				}
				res, err := udp.WriteToUDP(msg.Encode(), addr)
				if err != nil {
					fmt.Println("Failed to send response:", err)
				}

			}
		} else {
			for _, question := range receivedMsg.Questions {
				resMsg.Header.QDCOUNT++
				resMsg.Header.ANCOUNT++
				resMsg.Questions = append(resMsg.Questions, DNSQuestion{
					Name:  question.Name,
					Type:  1,
					Class: 1,
				})
				resMsg.Answers = append(resMsg.Answers, DNSAnswer{
					Name:     question.Name,
					Type:     1,
					Class:    1,
					TTL:      60,
					RDLENGTH: 4,
					RDATA:    0,
				})
			}
			fmt.Printf("Sending Message: %v\n", resMsg)
			// Create a response
			response = resMsg.Encode()
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

func Deserialize(data []byte) DNSMessage {
	header := deserializeHeader(data[:12])
	questionCount := int(header.QDCOUNT)
	fmt.Println(questionCount)
	counter := 0
	offset := 12
	var questions []DNSQuestion
	for counter < questionCount {
		question, newOffset := deserializeQuestion(data, offset)
		offset = newOffset
		questions = append(questions, question)
		counter++
	}
	return DNSMessage{
		Header:    deserializeHeader(data[:12]),
		Questions: questions,
	}
}

func deserializeHeader(data []byte) DNSHeader {
	return DNSHeader{
		ID: binary.BigEndian.Uint16(data[:2]),
		Flags: DNSHeaderFlags{
			QR:     (data[2] & 0x80) != 0,
			OPCODE: (data[2] >> 3) & 0x0F,
			AA:     (data[2] & 0x04) != 0,
			TC:     (data[2] & 0x02) != 0,
			RD:     (data[2] & 0x01) != 0,
			RA:     (data[2] & 0x80) != 0,
			Z:      (data[3] >> 4) & 0x07,
			RCODE:  data[3] & 0x0F,
		},
		QDCOUNT: binary.BigEndian.Uint16(data[4:6]),
		ANCOUNT: binary.BigEndian.Uint16(data[6:8]),
		NSCOUNT: binary.BigEndian.Uint16(data[8:10]),
		ARCOUNT: binary.BigEndian.Uint16(data[10:12]),
	}
}

func deserializeDomainName(data []byte, offset int) (string, int) {
	var name string
	for {
		labelLength := int(data[offset])
		offset++
		if labelLength == 0 {
			// End of domain name
			break
		}

		if labelLength >= 192 {
			// Pointer
			pointerOffset := (int(data[offset-1])&0x3F)<<8 + int(data[offset])
			offset++
			// Recursively resolve the pointer
			pointerName, _ := deserializeDomainName(data, pointerOffset)
			name += pointerName
			break
		}

		label := string(data[offset : offset+labelLength])
		if name != "" {
			name += "."
		}
		name += label
		offset += labelLength
	}
	return name, offset
}

func deserializeQuestion(data []byte, offset int) (DNSQuestion, int) {
	questionName, newOffset := deserializeDomainName(data, offset)
	offset = newOffset
	questionType := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	questionClass := binary.BigEndian.Uint16(data[offset : offset+2])
	return DNSQuestion{
		Name:  questionName,
		Type:  questionType,
		Class: questionClass,
	}, offset + 2
}
