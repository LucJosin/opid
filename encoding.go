package opid

import (
	"hash/crc32"
	"uuid"

	"github.com/lithammer/shortuuid/v5"
)

const (
	payloadAlphabet  = shortuuid.DefaultAlphabet
	checksumAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// [shortuuid] writes 22 Base57 characters.
	//
	// [shortuuid]: https://github.com/lithammer/shortuuid
	payloadWidth    = 22
	checksumSize    = 3
	radix           = len(checksumAlphabet)
	checksumModulus = radix * radix * radix

	bitsPerByte     = 8
	crcInitialValue = uint32(0xffffffff)
)

var payloadEncoder = shortuuid.NewEncoder(shortuuid.DefaultAlphabet)

func encodePayload(id uuid.UUID) string {
	return payloadEncoder.Encode(id)
}

func decodePayload(payload string) (uuid.UUID, error) {
	if len(payload) != payloadWidth {
		return uuid.UUID{}, ErrInvalidPayload
	}

	id, err := payloadEncoder.Decode(payload)
	if err != nil {
		return uuid.UUID{}, ErrInvalidPayload
	}
	return id, nil
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
		out[i] = checksumAlphabet[value%uint32(radix)]
		value /= uint32(radix)
	}

	return out
}

func checksum(body string) string {
	out := checksumBytes([]byte(body))
	return string(out[:])
}
