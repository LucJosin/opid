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
	sampleUUID          = "0199c4e2-7a3b-7c12-8e42-9f1234567890"
	testPayloadAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	testAlphabet        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	testPayloadWidth    = 22
	testChecksumSize    = 3
	testSeparatorCount  = 2
)

func newCodec(tb testing.TB) *opid.Codec {
	tb.Helper()
	codec, err := opid.NewCodec("usr")
	if err != nil {
		tb.Fatal(err)
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
	encoded := fixtureBase(new(big.Int).SetBytes(id[:]), testPayloadAlphabet)
	return strings.Repeat("2", testPayloadWidth-len(encoded)) + encoded
}

func fixtureBase62(value *big.Int) string {
	return fixtureBase(value, testAlphabet)
}

func fixtureBase(value *big.Int, alphabet string) string {
	n := new(big.Int).Set(value)
	if n.Sign() == 0 {
		return alphabet[:1]
	}
	var reversed []byte
	for n.Sign() > 0 {
		remainder := new(big.Int)
		n.QuoRem(n, big.NewInt(int64(len(alphabet))), remainder)
		reversed = append(reversed, alphabet[remainder.Int64()])
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return string(reversed)
}

func TestVectors(t *testing.T) {
	c := newCodec(t)

	tests := []struct {
		raw, payload, sum string
	}{
		{
			raw:     sampleUUID,
			payload: "2JFJNM9BCXm7nJtPJdJuB5",
			sum:     "lZw",
		},
		{
			raw:     "00000000-0000-7000-8000-000000000000",
			payload: "222222229tfndYm3pWg8fx",
			sum:     "2Iy",
		},
		{
			raw:     "ffffffff-ffff-7fff-bfff-ffffffffffff",
			payload: "oZEq7ovRSKw4RyseQUhKGL",
			sum:     "uvg",
		},
	}
	for _, v := range tests {
		t.Run(v.raw, func(t *testing.T) {
			id := uuid.MustParse(v.raw)
			want := "usr_" + v.payload + "_" + v.sum
			got, err := c.Encode(id)
			if err != nil || got != opid.ID(want) {
				t.Fatalf("Encode = %q, %v; want %q", got, err, want)
			}
			wantLength := len(c.Prefix()) + testSeparatorCount + testPayloadWidth + testChecksumSize
			if len(got) != wantLength {
				t.Fatalf("length = %d; want %d", len(got), wantLength)
			}
			decoded, err := c.Decode(got)
			if err != nil || decoded != id {
				t.Fatalf("Decode = %v, %v", decoded, err)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	validPrefixes := []string{
		"a",
		"usr",
		"ws",
		"dom",
		"a1",
		strings.Repeat("a", opid.MaxPrefixLength),
	}
	for _, prefix := range validPrefixes {
		t.Run(prefix, func(t *testing.T) {
			c, err := opid.NewCodec(prefix)
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

	invalidPrefixes := []string{
		"",
		"1usr",
		"Usr",
		"us_r",
		"us-r",
		"é",
		"u\xff",
		strings.Repeat("a", opid.MaxPrefixLength+1),
	}
	for _, prefix := range invalidPrefixes {
		if _, err := opid.NewCodec(prefix); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatalf("opid.NewCodec(%q): %v", prefix, err)
		}
	}

	c := newCodec(t)
	id, err := c.Encode(uuid.MustParse(sampleUUID))
	if err != nil {
		t.Fatal(err)
	}
	other, err := opid.NewCodec("dom")
	if err != nil {
		t.Fatal(err)
	}
	relabeledID := opid.ID("dom" + strings.TrimPrefix(id.String(), "usr"))
	if _, err := other.Decode(relabeledID); !errors.Is(err, opid.ErrInvalidChecksum) {
		t.Fatalf("relabel: %v", err)
	}
}

func TestDecodeErrors(t *testing.T) {
	c := newCodec(t)
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

	tests := []struct {
		name, input string
		want        error
	}{
		{
			name:  "empty",
			input: "",
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "missing checksum",
			input: "usr_" + payload,
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "separator",
			input: strings.ReplaceAll(good.String(), "_", "-"),
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "empty prefix",
			input: "_" + payload + "_" + sum,
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "empty payload",
			input: "usr__" + sum,
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "empty checksum",
			input: "usr_" + payload + "_",
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "extra component",
			input: good.String() + "_x",
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "prefix",
			input: "dom_" + payload + "_" + sum,
			want:  opid.ErrInvalidPrefix,
		},
		{
			name:  "uppercase prefix",
			input: "Usr_" + payload + "_" + sum,
			want:  opid.ErrInvalidPrefix,
		},
		{
			name:  "bad checksum",
			input: "usr_" + payload + "_" + alteredSum,
			want:  opid.ErrInvalidChecksum,
		},
		{
			name:  "checksum length",
			input: good.String() + "0",
			want:  opid.ErrInvalidChecksum,
		},
		{
			name:  "checksum syntax",
			input: "usr_" + payload + "_" + strings.Repeat("!", testChecksumSize),
			want:  opid.ErrInvalidChecksum,
		},
		{
			name:  "payload mutation",
			input: "usr_" + "3" + payload[1:] + "_" + sum,
			want:  opid.ErrInvalidChecksum,
		},
		{
			name:  "payload truncated",
			input: "usr_" + payload[1:] + "_" + sum,
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "payload truncated with fresh checksum",
			input: signedPayload("usr", payload[1:]),
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "payload case",
			input: "usr_" + strings.ToLower(payload) + "_" + sum,
			want:  opid.ErrInvalidChecksum,
		},
		{
			name:  "payload padded",
			input: signedPayload("usr", "2"+payload),
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "payload too long",
			input: signedPayload("usr", strings.Repeat("1", testPayloadWidth+1)),
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "bad alphabet",
			input: signedPayload("usr", "!"+payload[1:]),
			want:  opid.ErrInvalidPayload,
		},
		{
			name:  "excluded character",
			input: signedPayload("usr", "0"+payload[1:]),
			want:  opid.ErrInvalidPayload,
		},
		{
			name:  "overflow",
			input: signedPayload("usr", strings.Repeat("z", testPayloadWidth)),
			want:  opid.ErrInvalidPayload,
		},
		{
			name:  "overflow boundary",
			input: signedPayload("usr", "oZEq7ovRbLq6UnGMPwc8B6"),
			want:  opid.ErrInvalidPayload,
		},
		{
			name:  "zero",
			input: signedPayload("usr", fixtureUUIDPayload(uuid.Nil())),
			want:  opid.ErrInvalidUUID,
		},
		{
			name:  "max",
			input: signedPayload("usr", fixtureUUIDPayload(uuid.Max())),
			want:  opid.ErrInvalidUUID,
		},
		{
			name:  "too long",
			input: strings.Repeat("x", opid.MaxPrefixLength+testSeparatorCount+testPayloadWidth+testChecksumSize+1),
			want:  opid.ErrInvalidFormat,
		},
		{
			name:  "whitespace",
			input: " " + good.String(),
			want:  opid.ErrInvalidPrefix,
		},
		{
			name:  "prefix before checksum",
			input: "dom_" + payload + "_!!",
			want:  opid.ErrInvalidPrefix,
		},
		{
			name:  "alphabet before checksum",
			input: "usr_!" + payload[1:] + "_!!",
			want:  opid.ErrInvalidPayload,
		},
		{
			name:  "checksum before overflow",
			input: "usr_" + strings.Repeat("z", testPayloadWidth) + "_!!",
			want:  opid.ErrInvalidChecksum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := c.Decode(opid.ID(tt.input))
			if !errors.Is(err, tt.want) || id != (uuid.UUID{}) {
				t.Fatalf("Decode = %v, %v; want %v", id, err, tt.want)
			}
		})
	}
}

func TestUUIDBits(t *testing.T) {
	c := newCodec(t)
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
	invalidCodecs := []*opid.Codec{nil, {}}
	for _, c := range invalidCodecs {
		if _, err := c.Encode(uuid.MustParse(sampleUUID)); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatal(err)
		}
		if _, err := c.Decode(""); !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatal(err)
		}
	}
}

func TestDistinctUUIDs(t *testing.T) {
	c := newCodec(t)
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
	c := newCodec(t)
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
	codecs := []*opid.Codec{newCodec(t), nil, {}}

	for _, codec := range codecs {
		inputs := []string{
			"usr_2JFJNM9BCXm7nJtPJdJuB5_lZw",
			"",
			"ws_2JFJNM9BCXm7nJtPJdJuB5_lZw",
			"usr_2JFJNM9BCXm7nJtPJdJuB5_!!!",
			signedPayload("usr", "oZEq7ovRbLq6UnGMPwc8B6"),
			signedPayload("usr", fixtureUUIDPayload(uuid.Nil())),
		}

		for _, input := range inputs {
			want, wantErr := codec.Decode(opid.ID(input))
			got, err := codec.DecodeString(input)
			if got != want || err != wantErr {
				t.Fatalf("DecodeString(%q) = %v, %v; want %v, %v", input, got, err, want, wantErr)
			}
		}
	}
}

func TestEncodeString(t *testing.T) {
	codec := newCodec(t)

	validInputs := []string{
		sampleUUID,
		strings.ToUpper(sampleUUID),
		strings.ReplaceAll(sampleUUID, "-", ""),
		"{" + sampleUUID + "}",
		"urn:uuid:" + sampleUUID,
	}
	for _, input := range validInputs {
		want, err := codec.Encode(uuid.MustParse(sampleUUID))
		if err != nil {
			t.Fatal(err)
		}

		got, err := codec.EncodeString(input)
		if err != nil || got != want {
			t.Fatalf("EncodeString(%q) = %q, %v; want %q", input, got, err, want)
		}
	}

	invalidInputs := []string{
		"",
		"bad",
		" " + sampleUUID,
		"0199c4e2-7a3b-4c12-8e42-9f1234567890",
		"0199c4e2-7a3b-7c12-ce42-9f1234567890",
		uuid.Nil().String(),
		uuid.Max().String(),
	}
	for _, input := range invalidInputs {
		got, err := codec.EncodeString(input)
		if !got.IsZero() || !errors.Is(err, opid.ErrInvalidUUID) {
			t.Fatalf("EncodeString(%q) = %q, %v", input, got, err)
		}
	}

	invalidCodecs := []*opid.Codec{nil, {}}
	for _, codec := range invalidCodecs {
		inputs := []string{sampleUUID, "bad"}

		for _, input := range inputs {
			got, err := codec.EncodeString(input)
			if !got.IsZero() || !errors.Is(err, opid.ErrInvalidPrefix) {
				t.Fatalf("invalid codec: %q, %v", got, err)
			}
		}
	}
}

func TestNewID(t *testing.T) {
	prefixes := []string{"usr", "ws", strings.Repeat("a", opid.MaxPrefixLength)}
	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			codec, err := opid.NewCodec(prefix)
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

	invalidCodecs := []*opid.Codec{nil, {}}

	for _, codec := range invalidCodecs {
		id, err := codec.NewID()
		if !id.IsZero() || !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatalf("invalid codec: %q, %v", id, err)
		}
	}
}

func TestCodecPrefix(t *testing.T) {
	prefixes := []string{"usr", "ws", strings.Repeat("a", opid.MaxPrefixLength)}
	for _, prefix := range prefixes {
		codec, err := opid.NewCodec(prefix)
		if err != nil {
			t.Fatal(err)
		}
		if got := codec.Prefix(); got != prefix {
			t.Fatalf("Prefix = %q; want %q", got, prefix)
		}
	}

	invalidCodecs := []*opid.Codec{nil, {}}
	for _, codec := range invalidCodecs {
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

	codec := newCodec(t)
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
