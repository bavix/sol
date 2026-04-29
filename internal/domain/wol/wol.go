package wol

import "bytes"

const (
	PortDefault = 9
	BufferSize  = 2048
	HeaderSize  = 6
	RepeatCount = 16
	MACSize     = 6
	MagicByte   = 0xFF
)

func BuildMagicPacket(mac []byte) []byte {
	pkt := make([]byte, HeaderSize+RepeatCount*MACSize)

	for i := range HeaderSize {
		pkt[i] = MagicByte
	}

	offset := HeaderSize
	for range RepeatCount {
		copy(pkt[offset:offset+MACSize], mac)
		offset += MACSize
	}

	return pkt
}

func ContainsMagicPacket(payload []byte, expected []byte) bool {
	if len(payload) < len(expected) {
		return false
	}

	return bytes.Contains(payload, expected)
}

func ValidateMagicPacket(payload []byte, mac []byte) bool {
	if len(payload) < HeaderSize+RepeatCount*MACSize {
		return false
	}

	for i := range HeaderSize {
		if payload[i] != MagicByte {
			return false
		}
	}

	offset := HeaderSize
	for range RepeatCount {
		if !bytes.Equal(payload[offset:offset+MACSize], mac) {
			return false
		}

		offset += MACSize
	}

	return true
}
