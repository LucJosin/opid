package opid_test

import (
	"errors"
	"hash/crc32"
	"math/big"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
)

const (
	sampleUUID         = "0199c4e2-7a3b-7c12-8e42-9f1234567890"
	testAlphabet       = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	testPayloadWidth   = 22
	testChecksumSize   = 3
	testSeparatorCount = 2
)

func testCodec(t testing.TB) *opid.Codec {
	t.Helper()
	codec, err := opid.New("usr")
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func signedPayload(prefix, payload string) string {
	body := prefix + "_" + payload
	// Derive fixtures from the documented CRC rule, not private package helpers.
	value := crc32.ChecksumIEEE([]byte(body)) % 238328
	sum := fixtureBase62(new(big.Int).SetUint64(uint64(value)))
	sum = strings.Repeat("0", testChecksumSize-len(sum)) + sum
	return body + "_" + sum
}

func fixtureUUIDPayload(id uuid.UUID) string {
	return fixtureBase62(new(big.Int).SetBytes(id[:]))
}

func fixtureBase62(value *big.Int) string {
	n := new(big.Int).Set(value)
	if n.Sign() == 0 {
		return "0"
	}
	var reversed []byte
	for n.Sign() > 0 {
		remainder := new(big.Int)
		n.QuoRem(n, big.NewInt(int64(len(testAlphabet))), remainder)
		reversed = append(reversed, testAlphabet[remainder.Int64()])
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return string(reversed)
}

func TestVectors(t *testing.T) {
	c := testCodec(t)
	for _, v := range []struct{ raw, payload, sum string }{
		{sampleUUID, "31GQdK9BRjzTx1t1gYPlQ", "ccn"},
		{"00000000-0000-7000-8000-000000000000", "2dwHTRTFRxWLTM", "bav"},
		{"ffffffff-ffff-7fff-bfff-ffffffffffff", "7n42DGM5QePXFZlRJ5QKHn", "NMr"},
	} {
		t.Run(v.raw, func(t *testing.T) {
			id := uuid.MustParse(v.raw)
			want := "usr_" + v.payload + "_" + v.sum
			got, err := c.Encode(id)
			if err != nil || got != opid.ID(want) {
				t.Fatalf("Encode = %q, %v; want %q", got, err, want)
			}
			decoded, err := c.Decode(got)
			if err != nil || decoded != id {
				t.Fatalf("Decode = %v, %v", decoded, err)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	for _, prefix := range []string{"a", "usr", "ws", "dom", "a1", strings.Repeat("a", opid.MaxPrefixLength)} {
		t.Run(prefix, func(t *testing.T) {
			c, err := opid.New(prefix)
			if err != nil {
				t.Fatal(err)
			}
			id := uuid.MustParse("ffffffff-ffff-7fff-bfff-ffffffffffff")
			s, err := c.Encode(id)
			if err != nil {
				t.Fatal(err)
			}
			if len(s) != len(prefix)+testPayloadWidth+testChecksumSize+testSeparatorCount {
				t.Fatal("unexpected length")
			}
			got, err := c.Decode(s)
			if err != nil || got != id {
				t.Fatalf("round trip: %v", err)
			}
		})
	}
	for _, prefix := range []string{"", "1usr", "Usr", "us_r", "us-r", "é", "u\xff", strings.Repeat("a", opid.MaxPrefixLength+1)} {
		if _, err := opid.New(prefix); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatalf("opid.New(%q): %v", prefix, err)
		}
	}
	c := testCodec(t)
	id, err := c.Encode(uuid.MustParse(sampleUUID))
	if err != nil {
		t.Fatal(err)
	}
	other, err := opid.New("dom")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Decode(opid.ID("dom" + strings.TrimPrefix(id.String(), "usr"))); !errors.Is(err, opid.ErrInvalidChecksum) {
		t.Fatalf("relabel: %v", err)
	}
}

func TestDecodeErrors(t *testing.T) {
	c := testCodec(t)
	good, err := c.Encode(uuid.MustParse(sampleUUID))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(good.String(), "_")
	payload, sum := parts[1], parts[2]
	alteredSum := "0" + sum[1:]
	if sum[0] == '0' {
		alteredSum = "1" + sum[1:]
	}
	for _, tt := range []struct {
		name, input string
		want        error
	}{
		{"empty", "", opid.ErrInvalidFormat},
		{"missing checksum", "usr_" + payload, opid.ErrInvalidFormat},
		{"separator", strings.ReplaceAll(good.String(), "_", "-"), opid.ErrInvalidFormat},
		{"empty prefix", "_" + payload + "_" + sum, opid.ErrInvalidFormat},
		{"empty payload", "usr__" + sum, opid.ErrInvalidFormat},
		{"empty checksum", "usr_" + payload + "_", opid.ErrInvalidFormat},
		{"extra component", good.String() + "_x", opid.ErrInvalidFormat},
		{"prefix", "dom_" + payload + "_" + sum, opid.ErrInvalidPrefix},
		{"uppercase prefix", "Usr_" + payload + "_" + sum, opid.ErrInvalidPrefix},
		{"bad checksum", "usr_" + payload + "_" + alteredSum, opid.ErrInvalidChecksum},
		{"checksum length", good.String() + "0", opid.ErrInvalidChecksum},
		{"checksum syntax", "usr_" + payload + "_" + strings.Repeat("!", testChecksumSize), opid.ErrInvalidChecksum},
		{"payload mutation", "usr_" + "1" + payload[1:] + "_" + sum, opid.ErrInvalidChecksum},
		{"payload truncated", "usr_" + payload[1:] + "_" + sum, opid.ErrInvalidChecksum},
		{"payload case", "usr_" + strings.ToLower(payload) + "_" + sum, opid.ErrInvalidChecksum},
		{"payload padded", signedPayload("usr", "0"+payload), opid.ErrInvalidBase62},
		{"payload too long", signedPayload("usr", strings.Repeat("1", testPayloadWidth+1)), opid.ErrInvalidFormat},
		{"bad alphabet", signedPayload("usr", "!"+payload[1:]), opid.ErrInvalidBase62},
		{"overflow", signedPayload("usr", strings.Repeat("z", testPayloadWidth)), opid.ErrInvalidBase62},
		{"overflow boundary", signedPayload("usr", "7n42DGM5Tflk9n8mt7Fhc8"), opid.ErrInvalidBase62},
		{"zero", signedPayload("usr", "0"), opid.ErrInvalidUUID},
		{"max", signedPayload("usr", fixtureUUIDPayload(uuid.Max())), opid.ErrInvalidUUID},
		{"too long", strings.Repeat("x", opid.MaxPrefixLength+testSeparatorCount+testPayloadWidth+testChecksumSize+1), opid.ErrInvalidFormat},
		{"whitespace", " " + good.String(), opid.ErrInvalidPrefix},
		{"prefix before checksum", "dom_" + payload + "_!!", opid.ErrInvalidPrefix},
		{"alphabet before checksum", "usr_!" + payload[1:] + "_!!", opid.ErrInvalidBase62},
		{"checksum before overflow", "usr_" + strings.Repeat("z", testPayloadWidth) + "_!!", opid.ErrInvalidChecksum},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, err := c.Decode(opid.ID(tt.input))
			if !errors.Is(err, tt.want) || id != (uuid.UUID{}) {
				t.Fatalf("Decode = %v, %v; want %v", id, err, tt.want)
			}
		})
	}
}

func TestUUIDBits(t *testing.T) {
	c := testCodec(t)
	for version := range 16 {
		for variant := range 4 {
			id := uuid.MustParse(sampleUUID)
			id[6] = byte(version)<<4 | id[6]&15
			id[8] = byte(variant)<<6 | id[8]&63
			valid := version == 7 && variant == 2
			s, err := c.Encode(id)
			if valid {
				if err != nil {
					t.Fatal(err)
				}
				continue
			}
			if s != "" || !errors.Is(err, opid.ErrInvalidUUID) {
				t.Fatalf("version %d variant %d: %v", version, variant, err)
			}
			got, err := c.Decode(opid.ID(signedPayload("usr", fixtureUUIDPayload(id))))
			if got != (uuid.UUID{}) || !errors.Is(err, opid.ErrInvalidUUID) {
				t.Fatalf("Decode: %v", err)
			}
		}
	}
}

func TestInvalidCodec(t *testing.T) {
	for _, c := range []*opid.Codec{nil, {}} {
		if _, err := c.Encode(uuid.MustParse(sampleUUID)); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatal(err)
		}
		if _, err := c.Decode(""); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatal(err)
		}
	}
}

func TestDistinctUUIDs(t *testing.T) {
	c := testCodec(t)
	seen := make(map[opid.ID]bool)
	for i := range 1000 {
		id := uuid.MustParse(sampleUUID)
		id[14], id[15] = byte(i>>8), byte(i)
		s, err := c.Encode(id)
		if err != nil {
			t.Fatal(err)
		}
		if seen[s] {
			t.Fatal("duplicate encoding")
		}
		seen[s] = true
	}
}

func TestConcurrentRoundTrip(t *testing.T) {
	c := testCodec(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for i := range 100 {
				id := uuid.MustParse(sampleUUID)
				id[15] = byte(i)
				s, err := c.Encode(id)
				if err != nil {
					t.Error(err)
					return
				}
				got, err := c.Decode(s)
				if err != nil || got != id {
					t.Errorf("round trip: %v", err)
				}
				canonical, err := c.Encode(got)
				if err != nil || canonical != s {
					t.Error("noncanonical round trip")
				}
			}
		})
	}
	wg.Wait()
}

func TestDecodeString(t *testing.T) {
	for _, codec := range []*opid.Codec{testCodec(t), nil, {}} {
		for _, input := range []string{
			"usr_31GQdK9BRjzTx1t1gYPlQ_ccn",
			"",
			"ws_31GQdK9BRjzTx1t1gYPlQ_ccn",
			"usr_31GQdK9BRjzTx1t1gYPlQ_!!!",
			signedPayload("usr", "7n42DGM5Tflk9n8mt7Fhc8"),
			signedPayload("usr", "0"),
		} {
			want, wantErr := codec.Decode(opid.ID(input))
			got, err := codec.DecodeString(input)
			if got != want || err != wantErr {
				t.Fatalf("DecodeString(%q) = %v, %v; want %v, %v", input, got, err, want, wantErr)
			}
		}
	}
}

func TestEncodeString(t *testing.T) {
	codec := testCodec(t)
	for _, input := range []string{
		sampleUUID,
		strings.ToUpper(sampleUUID),
		strings.ReplaceAll(sampleUUID, "-", ""),
		"{" + sampleUUID + "}",
		"urn:uuid:" + sampleUUID,
	} {
		want, err := codec.Encode(uuid.MustParse(sampleUUID))
		if err != nil {
			t.Fatal(err)
		}

		got, err := codec.EncodeString(input)
		if err != nil || got != want {
			t.Fatalf("EncodeString(%q) = %q, %v; want %q", input, got, err, want)
		}
	}

	for _, input := range []string{
		"",
		"bad",
		" " + sampleUUID,
		"0199c4e2-7a3b-4c12-8e42-9f1234567890",
		"0199c4e2-7a3b-7c12-ce42-9f1234567890",
		uuid.Nil().String(),
		uuid.Max().String(),
	} {
		got, err := codec.EncodeString(input)
		if !got.IsZero() || !errors.Is(err, opid.ErrInvalidUUID) {
			t.Fatalf("EncodeString(%q) = %q, %v", input, got, err)
		}
	}

	for _, codec := range []*opid.Codec{nil, {}} {
		for _, input := range []string{sampleUUID, "bad"} {
			got, err := codec.EncodeString(input)
			if !got.IsZero() || !errors.Is(err, opid.ErrInvalidPrefix) {
				t.Fatalf("invalid codec: %q, %v", got, err)
			}
		}
	}
}

func TestNewID(t *testing.T) {
	for _, prefix := range []string{"usr", "ws", strings.Repeat("a", opid.MaxPrefixLength)} {
		t.Run(prefix, func(t *testing.T) {
			codec, err := opid.New(prefix)
			if err != nil {
				t.Fatal(err)
			}

			seen := make(map[opid.ID]bool)
			for range 100 {
				publicID, err := codec.NewID()
				if err != nil {
					t.Fatal(err)
				}

				id, err := codec.Decode(publicID)
				if err != nil || id == uuid.Nil() {
					t.Fatalf("generated opid.ID: %v", err)
				}
				if id[6]>>4 != 7 || id[8]>>6 != 2 {
					t.Fatal("expected UUIDv7 with RFC variant")
				}
				if seen[publicID] {
					t.Fatal("duplicate generated opid.ID")
				}
				seen[publicID] = true

				canonical, err := codec.Encode(id)
				if err != nil || canonical != publicID {
					t.Fatal("noncanonical generated opid.ID")
				}
			}
		})
	}

	for _, codec := range []*opid.Codec{nil, {}} {
		id, err := codec.NewID()
		if !id.IsZero() || !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatalf("invalid codec: %q, %v", id, err)
		}
	}
}

func TestCodecPrefix(t *testing.T) {
	for _, prefix := range []string{"usr", "ws", strings.Repeat("a", opid.MaxPrefixLength)} {
		codec, err := opid.New(prefix)
		if err != nil {
			t.Fatal(err)
		}

		if got := codec.Prefix(); got != prefix {
			t.Fatalf("Prefix = %q; want %q", got, prefix)
		}
	}

	for _, codec := range []*opid.Codec{nil, {}} {
		if got := codec.Prefix(); got != "" {
			t.Fatalf("invalid codec prefix = %q", got)
		}
	}
}

func TestNewIDConcurrent(t *testing.T) {
	const (
		workers      = 8
		idsPerWorker = 100
	)

	codec := testCodec(t)
	ids := make(chan opid.ID, workers*idsPerWorker)
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			for range idsPerWorker {
				id, err := codec.NewID()
				if err != nil {
					t.Error(err)
					return
				}

				original, err := codec.Decode(id)
				if err != nil {
					t.Error(err)
					return
				}

				canonical, err := codec.Encode(original)
				if err != nil || canonical != id {
					t.Error("generated opid.ID did not round trip")
					return
				}

				ids <- id
			}
		})
	}

	group.Wait()
	close(ids)

	seen := make(map[opid.ID]bool)
	for id := range ids {
		if seen[id] {
			t.Error("duplicate generated opid.ID")
		}
		seen[id] = true
	}

	if len(seen) != workers*idsPerWorker {
		t.Fatalf("generated %d distinct IDs; want %d", len(seen), workers*idsPerWorker)
	}
}
