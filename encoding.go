package opid

import (
	"encoding/binary"
	"hash/crc32"
	"math/bits"
	"uuid"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// A 128 bit value needs at most 22 Base62 digits.
	payloadWidth    = 22
	checksumSize    = 3
	radix           = len(alphabet)
	checksumModulus = radix * radix * radix

	wordBytes       = 8
	bitsPerByte     = 8
	digitCount      = '9' - '0' + 1
	letterCount     = 'Z' - 'A' + 1
	crcInitialValue = uint32(0xffffffff)
)

func encodeBase62(id uuid.UUID) string {
	var out [payloadWidth]byte
	return string(encodePayload(&out, id))
}

func encodePayload(out *[payloadWidth]byte, id uuid.UUID) []byte {
	high := binary.BigEndian.Uint64(id[:wordBytes])
	low := binary.BigEndian.Uint64(id[wordBytes:])
	start := len(out)

	for {
		remainder := high % uint64(radix)
		high /= uint64(radix)
		low, remainder = bits.Div64(remainder, low, uint64(radix))

		start--
		out[start] = alphabet[remainder]

		if high == 0 && low == 0 {
			return out[start:]
		}
	}
}

func decodeBase62(payload string) (uuid.UUID, error) {
	var id uuid.UUID
	if len(payload) < minPayloadLength || len(payload) > payloadWidth {
		return id, ErrInvalidBase62
	}

	if len(payload) > minPayloadLength && payload[0] == '0' {
		return id, ErrInvalidBase62
	}

	var high, low uint64
	for i := range payload {
		digit := digit62(payload[i])
		if digit < 0 {
			return uuid.UUID{}, ErrInvalidBase62
		}

		carry, nextLow := bits.Mul64(low, uint64(radix))
		overflow, nextHigh := bits.Mul64(high, uint64(radix))

		nextHigh, highCarry := bits.Add64(nextHigh, carry, 0)
		nextLow, carry = bits.Add64(nextLow, uint64(digit), 0)
		nextHigh, digitCarry := bits.Add64(nextHigh, 0, carry)

		if overflow != 0 || highCarry != 0 || digitCarry != 0 {
			return uuid.UUID{}, ErrInvalidBase62
		}

		high, low = nextHigh, nextLow
	}

	binary.BigEndian.PutUint64(id[:wordBytes], high)
	binary.BigEndian.PutUint64(id[wordBytes:], low)

	return id, nil
}

func digit62(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'A' && b <= 'Z':
		return int(b-'A') + digitCount
	case b >= 'a' && b <= 'z':
		return int(b-'a') + digitCount + letterCount
	default:
		return -1
	}
}

func checksumBytes(body []byte) [checksumSize]byte {
	// Use the IEEE table directly so the caller's buffer stays on the stack.
	crc := crcInitialValue
	for _, b := range body {
		crc = crc32.IEEETable[byte(crc)^b] ^ (crc >> bitsPerByte)
	}

	value := ^crc % uint32(checksumModulus)
	var out [checksumSize]byte
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = alphabet[value%uint32(radix)]
		value /= uint32(radix)
	}

	return out
}

func checksum(body string) string {
	out := checksumBytes([]byte(body))
	return string(out[:])
}
