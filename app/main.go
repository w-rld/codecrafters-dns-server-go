package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
)

func main() {
	var resolver string
	flag.StringVar(&resolver, "resolver", "", "resolver address")
	flag.Parse()
	fmt.Printf("Resolver: %s\n", resolver)
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

		var responseMsg DNSMessage
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedMsg.ToString())

		// handle request from client
		rcode := 0
		if receivedMsg.Header.Flags.OPCODE != 0 {
			rcode = 4
		}
		responseMsg = DNSMessage{
			Header: DNSHeader{
				ID: receivedMsg.Header.ID,
				Flags: DNSHeaderFlags{
					QR:     false,
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
		for _, question := range receivedMsg.Questions {
			responseMsg.Header.QDCOUNT++
			responseMsg.Questions = append(responseMsg.Questions, DNSQuestion{
				Name:  question.Name,
				Type:  1,
				Class: 1,
			})
			if receivedMsg.Header.Flags.RD && resolver != "" {
				// send message to resolver
				forwardMsg := constructForwardMessage(responseMsg.Header.ID, responseMsg.Header.Flags.OPCODE, responseMsg.Header.Flags.RD, responseMsg.Header.Flags.RCODE, question)

				res, err := forwardMsg.forward(resolver)
				if err != nil {
					fmt.Println("Failed to forward:", err)
				}
				responseMsg.Header.Flags = res.Header.Flags
				if res.Header.ANCOUNT > 0 {
					responseMsg.Header.ANCOUNT++
					responseMsg.Answers = append(responseMsg.Answers, res.Answers[0])
				}
			} else {
				// answer message without resolver
				responseMsg.Header.Flags.QR = true
				responseMsg.Header.ANCOUNT = responseMsg.Header.QDCOUNT
				responseMsg.Answers = append(responseMsg.Answers, DNSAnswer{
					Name:     question.Name,
					Type:     1,
					Class:    1,
					TTL:      60,
					RDLENGTH: 4,
					RDATA:    0,
				})
			}
		}

		fmt.Printf("Sending Message: %s\n", responseMsg.ToString())
		res := responseMsg.Encode()
		_, err = udpConn.WriteToUDP(res, source)
		if err != nil {
			fmt.Println("Failed to send dataToSend:", err)
		}
	}
}

func constructForwardMessage(ID uint16, OPCODE uint8, RD bool, RCODE uint8, Question DNSQuestion) DNSMessage {
	return DNSMessage{
		Header: DNSHeader{
			ID: ID,
			Flags: DNSHeaderFlags{
				QR:     false,
				OPCODE: OPCODE,
				AA:     false,
				TC:     false,
				RD:     RD,
				RA:     false,
				Z:      0,
				RCODE:  RCODE,
			},
			QDCOUNT: 1,
			ANCOUNT: 0,
			NSCOUNT: 0,
			ARCOUNT: 0,
		},
		Questions: []DNSQuestion{Question},
		Answers:   []DNSAnswer{},
	}
}

func (message DNSMessage) forward(addr string) (DNSMessage, error) {
	var res DNSMessage

	conn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return res, err
	}
	defer conn.Close()

	fmt.Printf("Sending message to resolver: %s\n", message.ToString())
	_, err = conn.Write(message.Encode())
	if err != nil {
		return res, err
	}
	buf := make([]byte, 512)
	size, err := conn.Read(buf)
	res = Deserialize(buf)
	fmt.Printf("Received %d bytes from resolver: %s\n", size, res.ToString())
	return res, nil
}

func Deserialize(data []byte) DNSMessage {
	header := deserializeHeader(data[:12])
	questionCounter, answerCounter, offset, questionCount, answerCount := 0, 0, 12, int(header.QDCOUNT), int(header.ANCOUNT)
	var questions []DNSQuestion
	for questionCounter < questionCount {
		question, newOffset := deserializeQuestion(data, offset)
		offset = newOffset
		questions = append(questions, question)
		questionCounter++
	}
	var answers []DNSAnswer
	for answerCounter < answerCount {
		answer, newOffset := deserializeAnswer(data, offset)
		offset = newOffset
		answers = append(answers, answer)
		answerCounter++
	}
	return DNSMessage{
		Header:    deserializeHeader(data[:12]),
		Questions: questions,
		Answers:   answers,
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
		if name != "" {
			name += "."
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

func deserializeAnswer(data []byte, offset int) (DNSAnswer, int) {
	Name, newOffset := deserializeDomainName(data, offset)
	offset = newOffset
	Type := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	Class := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	TLL := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	RDLENGTH := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	RDATA := binary.BigEndian.Uint32(data[offset : offset+4])
	fmt.Printf("RDLENGTH: %d, RDATA: %d, Bytes: %b\n",RDLENGTH, RDATA, data[offset:offset+4])
	offset += 4
	return DNSAnswer{
		Name:     Name,
		Type:     Type,
		Class:    Class,
		TTL:      TLL,
		RDLENGTH: RDLENGTH,
		RDATA:    RDATA,
	}, offset
}
