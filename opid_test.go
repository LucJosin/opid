package opid_test

import (
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
)

func TestID(t *testing.T) {
	var empty opid.ID
	if !empty.IsZero() || empty.String() != "" {
		t.Fatal("unexpected zero ID")
	}
	codec, err := opid.New("usr")
	if err != nil {
		t.Fatal(err)
	}
	original := uuid.MustParse("0199c4e2-7a3b-7c12-8e42-9f1234567890")
	id, err := codec.Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	if id.IsZero() || id.String() != "usr_31GQdK9BRjzTx1t1gYPlQ_ccn" {
		t.Fatal("unexpected ID")
	}
	data, err := json.Marshal(struct {
		ID opid.ID `json:"id"`
	}{id})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"id":"usr_31GQdK9BRjzTx1t1gYPlQ_ccn"}` {
		t.Fatalf("JSON = %s", data)
	}
	var record struct {
		ID opid.ID `json:"id"`
	}
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
