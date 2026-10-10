package opid

import (
	"hash/crc32"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
	"uuid"
)

func TestBase62(t *testing.T) {
	tests := []struct {
		id      uuid.UUID
		payload string
	}{
		{
			id:      uuid.Nil(),
			payload: "0",
		},
		{
			id:      uuid.UUID{15: 1},
			payload: "1",
		},
		{
			id:      uuid.Max(),
			payload: "7n42DGM5Tflk9n8mt7Fhc7",
		},
	}
	for _, v := range tests {
		if got := encodeBase62(v.id); got != v.payload {
			t.Fatalf("encode = %q", got)
		}
		got, err := decodeBase62(v.payload)
		if err != nil || got != v.id {
			t.Fatalf("decode = %v, %v", got, err)
		}
	}

	invalidPayloads := []string{
		"",
		"01",
		strings.Repeat("0", payloadWidth+1),
		strings.Repeat("z", payloadWidth),
		"7n42DGM5Tflk9n8mt7Fhc8",
		strings.Repeat("!", payloadWidth),
	}
	for _, s := range invalidPayloads {
		if _, err := decodeBase62(s); err != ErrInvalidBase62 {
			t.Fatalf("decode(%q): %v", s, err)
		}
	}
}

func TestBase62Reference(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		var id uuid.UUID
		for i := range id {
			id[i] = byte(rng.Uint32())
		}
		expected := referenceBase62(new(big.Int).SetBytes(id[:]))
		got := encodeBase62(id)
		if got != expected {
			t.Fatal("reference mismatch")
		}
		decoded, err := decodeBase62(got)
		if err != nil || decoded != id {
			t.Fatal("round trip mismatch")
		}
	}
}

func TestBase62WordBoundaries(t *testing.T) {
	// Exercise carries at every bit, including the boundary between words.
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	for bit := 0; bit <= 128; bit++ {
		power := new(big.Int).Lsh(big.NewInt(1), uint(bit))
		deltas := []int64{-1, 0, 1}
		for _, delta := range deltas {
			n := new(big.Int).Add(power, big.NewInt(delta))
			payload := referenceBase62(n)
			id, err := decodeBase62(payload)
			if n.Cmp(limit) >= 0 {
				if err != ErrInvalidBase62 || id != (uuid.UUID{}) {
					t.Fatalf("overflow at bit %d delta %d: %v", bit, delta, err)
				}
				continue
			}
			if err != nil || new(big.Int).SetBytes(id[:]).Cmp(n) != 0 {
				t.Fatalf("decode at bit %d delta %d: %v", bit, delta, err)
			}
			if got := encodeBase62(id); got != payload {
				t.Fatalf("encode at bit %d delta %d: %q; want %q", bit, delta, got, payload)
			}
		}
	}
}

func referenceBase62(value *big.Int) string {
	n := new(big.Int).Set(value)
	if n.Sign() == 0 {
		return "0"
	}
	var reversed []byte
	for n.Sign() > 0 {
		remainder := new(big.Int)
		n.QuoRem(n, big.NewInt(62), remainder)
		reversed = append(reversed, alphabet[remainder.Int64()])
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return string(reversed)
}

func TestChecksum(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{
			input: "",
			want:  "000",
		},
		{
			input: "123456789",
			want:  "RME",
		},
		{
			input: "usr_31GQdK9BRjzTx1t1gYPlQ",
			want:  "ccn",
		},
		{
			input: "usr_2dwHTRTFRxWLTM",
			want:  "bav",
		},
		{
			input: "usr_7n42DGM5QePXFZlRJ5QKHn",
			want:  "NMr",
		},
		{
			input: "usr_0",
			want:  "b0r",
		},
	}
	for _, v := range tests {
		if got := checksum(v.input); got != v.want {
			t.Fatalf("checksum(%q) = %q; want %q", v.input, got, v.want)
		}
	}
}

func TestChecksumReference(t *testing.T) {
	const maxInputBytes = 1024
	rng := rand.New(rand.NewPCG(3, 4))
	for length := 0; length <= maxInputBytes; length++ {
		body := make([]byte, length)
		for i := range body {
			body[i] = byte(rng.Uint32())
		}

		got := checksumBytes(body)
		if want := referenceChecksum(body); string(got[:]) != want {
			t.Fatalf("checksum for length %d = %q; want %q", length, got, want)
		}
	}
}

func referenceChecksum(body []byte) string {
	// Use the documented modulus and width independently of the production helper.
	value := crc32.ChecksumIEEE(body) % 238328
	encoded := referenceBase62(new(big.Int).SetUint64(uint64(value)))
	return strings.Repeat("0", 3-len(encoded)) + encoded
}

const sampleUUID = "0199c4e2-7a3b-7c12-8e42-9f1234567890"

func newTestCodec(tb testing.TB) *Codec {
	tb.Helper()
	c, err := New("usr")
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

func signedPayload(prefix, payload string) string {
	body := prefix + "_" + payload
	return body + "_" + checksum(body)
}
