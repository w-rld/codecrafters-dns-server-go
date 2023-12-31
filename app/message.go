package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type DNSMessage struct {
	Header    DNSHeader
	Questions []DNSQuestion
	Answers   []DNSAnswer
}

func (message DNSMessage) Encode() []byte {
	result := message.Header.Encode()
	for _, question := range message.Questions {
		result = append(result, question.Encode()...)
	}
	for _, answer := range message.Answers {
		result = append(result, answer.Encode()...)
	}
	return result
}

type DNSAnswer struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLENGTH uint16
	RDATA uint32
}

func (answer DNSAnswer) Encode() []byte {
	questionAdd := make([]byte, 4)
	binary.BigEndian.PutUint16(questionAdd[:2], answer.Type)
	binary.BigEndian.PutUint16(questionAdd[2:4], answer.Class)
	result := append(domainToBytes(answer.Name), questionAdd...)
	answerData := make([]byte, 10)
	binary.BigEndian.PutUint32(answerData[:4], answer.TTL)
	binary.BigEndian.PutUint16(answerData[4:6], answer.RDLENGTH)
	binary.BigEndian.PutUint32(answerData[6:10], answer.RDATA)
	return append(result, answerData...)
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

func (question DNSQuestion) Encode() []byte {
	questionAdd := make([]byte, 4)
	binary.BigEndian.PutUint16(questionAdd[:2], question.Type)
	binary.BigEndian.PutUint16(questionAdd[2:4], question.Class)
	return append(domainToBytes(question.Name), questionAdd...)
}

func domainToBytes(domain string) []byte {
	var result []byte
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		result = append(result, byte(len(label)))
		result = append(result, []byte(label)...)
	}
	return append(result, 0x00)
}

type DNSHeaderFlags struct {
	QR     bool  // Query Response
	OPCODE uint8 // Operation Code
	AA     bool  // Authoritative Answer
	TC     bool  // Truncate Message
	RD     bool  // Recursive Desired
	RA     bool  // Recursion Available
	Z      uint8 // Reserved
	RCODE  uint8 // Response Code
}

type DNSHeader struct {
	ID      uint16 // Packet Identifier
	Flags   DNSHeaderFlags
	QDCOUNT uint16 // Question Count
	ANCOUNT uint16 // Answer Count
	NSCOUNT uint16 // Authority Count
	ARCOUNT uint16 // Additional Count
}

func (header DNSHeader) Encode() []byte {
	result := make([]byte, 12)

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

	binary.BigEndian.PutUint16(result[:2], header.ID)
	binary.BigEndian.PutUint16(result[2:4], flags)
	binary.BigEndian.PutUint16(result[4:6], header.QDCOUNT)
	binary.BigEndian.PutUint16(result[6:8], header.ANCOUNT)
	binary.BigEndian.PutUint16(result[8:10], header.NSCOUNT)
	binary.BigEndian.PutUint16(result[10:], header.ARCOUNT)

	return result
}

func (message DNSMessage) ToString() string {
	var questions string
	for _, q := range message.Questions {
		if questions != "" {
			questions += ","
		}
		questions += q.ToString()
	}
	var answers string
	for _, a := range message.Answers {
		if answers != "" {
			answers += ","
		}
		answers += a.ToString()
	}
	return fmt.Sprintf("DNSMessage[Header: %s, Questions: [%s], Answers: [%s]]", message.Header.ToString(), questions, answers)
}

func (h DNSHeader) ToString() string {
	return fmt.Sprintf("DNSHeader[ID: %d, Flags: %s, QDCOUNT: %d, ANCOUNT: %d, NSCOUNT: %d, ARCOUNT: %d]", h.ID, h.Flags.ToString(), h.QDCOUNT, h.ANCOUNT, h.NSCOUNT, h.ARCOUNT)
}

func (hf DNSHeaderFlags) ToString() string {
	return fmt.Sprintf("DNSHeaderFlags[QR: %t, OPCODE: %d, AA: %t, TC: %t, RD: %t, RA: %t]", hf.QR, hf.OPCODE, hf.AA, hf.TC, hf.RD, hf.RA)
}

func (q DNSQuestion) ToString() string {
	return fmt.Sprintf("DNSQuestion[Name: %s, Type: %d, Class: %d]", q.Name, q.Type, q.Class)
}

func (a DNSAnswer) ToString() string {
	return fmt.Sprintf("DNSAnswer[Name: %s, Type: %d, Class: %d, TTL: %d, RDLENGHT: %d, RDATA: %d]", a.Name, a.Type, a.Class, a.TTL, a.RDLENGTH, a.RDATA)
}