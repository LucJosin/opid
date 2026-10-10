package opid

import (
	"math/big"
	"strings"
	"testing"
	"uuid"
)

func FuzzRoundTrip(f *testing.F) {
	seed := uuid.MustParse(sampleUUID)
	f.Add(seed[:], "usr")
	f.Add(make([]byte, 16), strings.Repeat("a", MaxPrefixLength))
	f.Fuzz(func(t *testing.T, raw []byte, prefix string) {
		if len(raw) != 16 {
			return
		}
		c, err := New(prefix)
		if err != nil {
			return
		}
		var id uuid.UUID
		copy(id[:], raw)
		id[6] = id[6]&0x0f | 0x70
		id[8] = id[8]&0x3f | 0x80
		s, err := c.Encode(id)
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Decode(s)
		if err != nil || got != id {
			t.Fatalf("round trip: %v", err)
		}
		canonical, err := c.Encode(got)
		if err != nil || canonical != s {
			t.Fatal("noncanonical round trip")
		}
	})
}

func FuzzDecode(f *testing.F) {
	f.Add("usr_31GQdK9BRjzTx1t1gYPlQ_ccn")
	f.Add("")
	f.Add("usr_zzzzzzzzzzzzzzzzzzzzzz_000")
	c := newTestCodec(f)
	f.Fuzz(func(t *testing.T, s string) { checkDecode(t, c, s) })
}

func FuzzPayload(f *testing.F) {
	payloads := []string{
		"0",
		"31GQdK9BRjzTx1t1gYPlQ",
		"7n42DGM5Tflk9n8mt7Fhc8",
		"zzzzzzzzzzzzzzzzzzzzzz",
	}
	for _, payload := range payloads {
		f.Add(payload)
	}
	c := newTestCodec(f)
	f.Fuzz(func(t *testing.T, payload string) {
		if len(payload) > payloadWidth {
			return
		}
		checkDecode(t, c, signedPayload("usr", payload))
	})
}

func checkDecode(t *testing.T, c *Codec, s string) {
	t.Helper()
	id, err := c.Decode(ID(s))
	if err != nil {
		if id != (uuid.UUID{}) {
			t.Fatal("nonzero UUID on error")
		}
		return
	}
	canonical, err := c.Encode(id)
	if err != nil || canonical.String() != s {
		t.Fatal("accepted noncanonical ID")
	}
}

func FuzzBase62(f *testing.F) {
	seeds := []uuid.UUID{
		uuid.Nil(),
		uuid.Max(),
		{7: 0xff, 8: 0xff},
		{8: 1},
		uuid.MustParse(sampleUUID),
	}
	for _, seed := range seeds {
		f.Add(seed[:])
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		var id uuid.UUID
		if len(raw) != len(id) {
			return
		}
		copy(id[:], raw)

		payload := encodeBase62(id)
		expected := referenceBase62(new(big.Int).SetBytes(raw))
		if payload != expected {
			t.Fatalf("payload = %q; want %q", payload, expected)
		}

		decoded, err := decodeBase62(payload)
		if err != nil || decoded != id {
			t.Fatalf("Base62 round trip: %v", err)
		}
	})
}

func FuzzChecksum(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("123456789"))
	f.Add([]byte("usr_31GQdK9BRjzTx1t1gYPlQ"))
	f.Add([]byte{0, 0xff, 0x80})
	f.Fuzz(func(t *testing.T, body []byte) {
		const maxInputBytes = 4096
		if len(body) > maxInputBytes {
			return
		}

		got := checksumBytes(body)
		if want := referenceChecksum(body); string(got[:]) != want {
			t.Fatalf("checksum = %q; want %q", got, want)
		}
	})
}
