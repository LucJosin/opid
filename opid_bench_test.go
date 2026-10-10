package opid

import (
	"strings"
	"testing"
	"uuid"
)

func benchmarkCodec(b *testing.B, prefix string) (*Codec, uuid.UUID, ID) {
	b.Helper()
	c, err := New(prefix)
	if err != nil {
		b.Fatal(err)
	}
	id := uuid.MustParse(sampleUUID)
	s, err := c.Encode(id)
	if err != nil {
		b.Fatal(err)
	}
	if got, err := c.Decode(s); err != nil || got != id {
		b.Fatal("invalid benchmark fixture")
	}
	return c, id, s
}

func BenchmarkEncode(b *testing.B) {
	prefixes := []string{"usr", strings.Repeat("a", MaxPrefixLength)}
	for _, prefix := range prefixes {
		b.Run(prefix, func(b *testing.B) {
			c, id, _ := benchmarkCodec(b, prefix)
			b.ReportAllocs()
			for b.Loop() {
				_, _ = c.Encode(id)
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	prefixes := []string{"usr", strings.Repeat("a", MaxPrefixLength)}
	for _, prefix := range prefixes {
		b.Run(prefix, func(b *testing.B) {
			c, _, s := benchmarkCodec(b, prefix)
			b.ReportAllocs()
			for b.Loop() {
				_, _ = c.Decode(s)
			}
		})
	}
}

func BenchmarkEncodeParallel(b *testing.B) {
	c, id, _ := benchmarkCodec(b, "usr")
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Encode(id)
		}
	})
}

func BenchmarkDecodeParallel(b *testing.B) {
	c, _, s := benchmarkCodec(b, "usr")
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Decode(s)
		}
	})
}

func BenchmarkDecodeInvalid(b *testing.B) {
	c := newTestCodec(b)

	tests := []struct {
		name, s string
		want    error
	}{
		{
			name: "format",
			s:    "bad",
			want: ErrInvalidFormat,
		},
		{
			name: "checksum",
			s:    "usr_31GQdK9BRjzTx1t1gYPlQ_!!!",
			want: ErrInvalidChecksum,
		},
		{
			name: "overflow",
			s:    signedPayload("usr", "7n42DGM5Tflk9n8mt7Fhc8"),
			want: ErrInvalidBase62,
		},
	}
	for _, v := range tests {
		b.Run(v.name, func(b *testing.B) {
			if _, err := c.Decode(ID(v.s)); err != v.want {
				b.Fatalf("fixture: %v", err)
			}
			b.ReportAllocs()
			for b.Loop() {
				_, _ = c.Decode(ID(v.s))
			}
		})
	}
}

func BenchmarkBase62Encode(b *testing.B) {
	id := uuid.MustParse(sampleUUID)
	b.ReportAllocs()
	for b.Loop() {
		_ = encodeBase62(id)
	}
}

func BenchmarkBase62Decode(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = decodeBase62("31GQdK9BRjzTx1t1gYPlQ")
	}
}

func BenchmarkChecksum(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = checksumBytes([]byte("usr_31GQdK9BRjzTx1t1gYPlQ"))
	}
}

func BenchmarkDecodeString(b *testing.B) {
	c, _, id := benchmarkCodec(b, "usr")
	input := id.String()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.DecodeString(input)
	}
}

func BenchmarkEncodeString(b *testing.B) {
	c, _, _ := benchmarkCodec(b, "usr")
	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.EncodeString(sampleUUID)
	}
}
