package main

import (
    "encoding/binary"
    "strings"
)

type DNSMessage struct {
    Header  DNSHeader
    Question DNSQuestion
}

func (message DNSMessage) Encode() []byte {
    headerBytes := message.Header.Encode()
    questionBytes := message.Question.Encode()
    return append(headerBytes, questionBytes...)
}

type DNSQuestion struct {
    Name string
    Type uint16
    Class uint16
}

func (question DNSQuestion) Encode() []byte {
    var result []byte
    labels := strings.Split(question.Name, ".")
    for _, label := range labels {
        result = append(result, byte(len(label)))
      result = append(result, []byte(label)...)
    }
    result = append(result, 0x00)
    result = append(result, byte(question.Type>>8), byte(question.Type))
    result = append(result, byte(question.Class>>8), byte(question.Class))
    return result
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

type DNSHeader struct {
    ID uint16 // Packet Identifier
    Flags DNSHeaderFlags
    QDCOUNT uint16 // Question Count
    ANCOUNT uint16 // Answer Count
    NSCOUNT uint16 // Authority Count
    ARCOUNT uint16 // Additional Count
}

func (header DNSHeader) Encode() []byte {
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