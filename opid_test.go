package opid_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
)

func TestNew(t *testing.T) {
	codec, err := opid.NewCodec("usr")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[opid.ID]bool)
	for range 100 {
		id, err := opid.New("usr")
		if err != nil || id.IsZero() {
			t.Fatalf("New = %q, %v", id, err)
		}
		original, err := codec.Decode(id)
		if err != nil || original[6]>>4 != 7 || original[8]>>6 != 2 {
			t.Fatalf("expected UUIDv7: %v", err)
		}
		if seen[id] {
			t.Fatal("duplicate generated ID")
		}
		seen[id] = true
	}

	invalidPrefixes := []string{"", "Usr", "1usr", "us_r"}
	for _, prefix := range invalidPrefixes {
		id, err := opid.New(prefix)
		if !id.IsZero() || !errors.Is(err, opid.ErrInvalidPrefix) {
			t.Fatalf("New(%q) = %q, %v", prefix, id, err)
		}
	}
}

func TestID(t *testing.T) {
	type idRecord struct {
		ID opid.ID `json:"id"`
	}

	var empty opid.ID
	if !empty.IsZero() || empty.String() != "" {
		t.Fatal("unexpected zero ID")
	}
	codec, err := opid.NewCodec("usr")
	if err != nil {
		t.Fatal(err)
	}
	original := uuid.MustParse("0199c4e2-7a3b-7c12-8e42-9f1234567890")
	id, err := codec.Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	if id.IsZero() || id.String() != "usr_2JFJNM9BCXm7nJtPJdJuB5_lZw" {
		t.Fatal("unexpected ID")
	}
	data, err := json.Marshal(idRecord{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"id":"usr_2JFJNM9BCXm7nJtPJdJuB5_lZw"}` {
		t.Fatalf("JSON = %s", data)
	}
	var record idRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(record.ID)
	if err != nil || decoded != original {
		t.Fatalf("round trip: %v", err)
	}
	// Conversion and JSON decoding do not establish validity.
	if err := json.Unmarshal([]byte(`{"id":"bad"}`), &record); err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Decode(record.ID); !errors.Is(err, opid.ErrInvalidFormat) {
		t.Fatalf("invalid ID: %v", err)
	}
}

func TestIDPrefix(t *testing.T) {
	tests := []struct {
		name string
		id   opid.ID
		want string
	}{
		{
			name: "valid ID",
			id:   "usr_2JFJNM9BCXm7nJtPJdJuB5_lZw",
			want: "usr",
		},
		{
			name: "unchecked remainder",
			id:   "usr_bad",
			want: "usr",
		},
		{
			name: "empty remainder",
			id:   "usr_",
			want: "usr",
		},
		{
			name: "maximum prefix",
			id:   opid.ID(strings.Repeat("a", opid.MaxPrefixLength) + "_bad"),
			want: strings.Repeat("a", opid.MaxPrefixLength),
		},
		{
			name: "zero ID",
			id:   "",
			want: "",
		},
		{
			name: "missing separator",
			id:   "usr",
			want: "",
		},
		{
			name: "empty prefix",
			id:   "_bad",
			want: "",
		},
		{
			name: "uppercase prefix",
			id:   "Usr_bad",
			want: "",
		},
		{
			name: "digit first",
			id:   "1usr_bad",
			want: "",
		},
		{
			name: "invalid character",
			id:   "us-r_bad",
			want: "",
		},
		{
			name: "prefix too long",
			id:   opid.ID(strings.Repeat("a", opid.MaxPrefixLength+1) + "_bad"),
			want: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.id.Prefix(); got != test.want {
				t.Fatalf("Prefix = %q; want %q", got, test.want)
			}
		})
	}
}
