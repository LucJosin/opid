package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
)

const (
	sampleUUID = "0199c4e2-7a3b-7c12-8e42-9f1234567890"
	sampleID   = "usr_31GQdK9BRjzTx1t1gYPlQ_ccn"
)

func TestEndpoints(t *testing.T) {
	handler := testHandler(t)
	for _, tt := range []struct {
		name, method, path string
		status             int
		id, rawUUID        string
	}{
		{"encode", "GET", "/usr/encode/" + sampleUUID, http.StatusOK, sampleID, ""},
		{"decode", "GET", "/usr/decode/" + sampleID, http.StatusOK, "", sampleUUID},
		{"wrong prefix", "GET", "/usr/decode/ws_31GQdK9BRjzTx1t1gYPlQ_ccn", http.StatusBadRequest, "", ""},
		{"unsupported resource", "GET", "/ws/generate", http.StatusNotFound, "", ""},
		{"invalid UUID", "GET", "/usr/encode/bad", http.StatusBadRequest, "", ""},
		{"UUIDv4", "GET", "/usr/encode/0199c4e2-7a3b-4c12-8e42-9f1234567890", http.StatusBadRequest, "", ""},
		{"bad checksum", "GET", "/usr/decode/usr_31GQdK9BRjzTx1t1gYPlQ_!!!", http.StatusBadRequest, "", ""},
		{"missing UUID", "GET", "/usr/encode/", http.StatusNotFound, "", ""},
		{"missing ID", "GET", "/usr/decode/", http.StatusNotFound, "", ""},
		{"method", "POST", "/usr/generate", http.StatusMethodNotAllowed, "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tt.status {
				t.Fatalf("status = %d; body = %s", recorder.Code, recorder.Body.String())
			}
			if tt.status == http.StatusMethodNotAllowed || tt.status == http.StatusNotFound {
				return
			}
			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatal("expected JSON")
			}
			var body response
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ID.String() != tt.id || body.UUID != tt.rawUUID {
				t.Fatalf("unexpected response: %+v", body)
			}
			if tt.status >= http.StatusBadRequest && body.Error == "" {
				t.Fatal("missing error")
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	handler := testHandler(t)
	codec, err := opid.New("usr")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[opid.ID]bool)
	for range 2 {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/usr/generate", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d", recorder.Code)
		}
		if recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("generated IDs must not be cached")
		}

		var body response
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		original, err := uuid.Parse(body.UUID)
		if err != nil || original == uuid.Nil() {
			t.Fatalf("generated UUID: %q, %v", body.UUID, err)
		}
		id, err := codec.Decode(body.ID)
		if err != nil || id != original {
			t.Fatalf("generated ID: %v", err)
		}
		if seen[body.ID] {
			t.Fatal("duplicate generated ID")
		}
		seen[body.ID] = true
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	codec, err := opid.New(resourcePrefix)
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(codec)
}
