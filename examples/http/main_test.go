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
	sampleID   = "usr_2JFJNM9BCXm7nJtPJdJuB5_lZw"
)

func TestEndpoints(t *testing.T) {
	handler := newTestHandler(t)

	tests := []struct {
		name, method, path string
		status             int
		id, rawUUID        string
	}{
		{
			name:    "encode",
			method:  "GET",
			path:    "/usr/encode/" + sampleUUID,
			status:  http.StatusOK,
			id:      sampleID,
			rawUUID: "",
		},
		{
			name:    "decode",
			method:  "GET",
			path:    "/usr/decode/" + sampleID,
			status:  http.StatusOK,
			id:      "",
			rawUUID: sampleUUID,
		},
		{
			name:    "wrong prefix",
			method:  "GET",
			path:    "/usr/decode/ws_2JFJNM9BCXm7nJtPJdJuB5_lZw",
			status:  http.StatusBadRequest,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "unsupported resource",
			method:  "GET",
			path:    "/ws/generate",
			status:  http.StatusNotFound,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "invalid UUID",
			method:  "GET",
			path:    "/usr/encode/bad",
			status:  http.StatusBadRequest,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "UUIDv4",
			method:  "GET",
			path:    "/usr/encode/0199c4e2-7a3b-4c12-8e42-9f1234567890",
			status:  http.StatusBadRequest,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "bad checksum",
			method:  "GET",
			path:    "/usr/decode/usr_2JFJNM9BCXm7nJtPJdJuB5_!!!",
			status:  http.StatusBadRequest,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "missing UUID",
			method:  "GET",
			path:    "/usr/encode/",
			status:  http.StatusNotFound,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "missing ID",
			method:  "GET",
			path:    "/usr/decode/",
			status:  http.StatusNotFound,
			id:      "",
			rawUUID: "",
		},
		{
			name:    "method",
			method:  "POST",
			path:    "/usr/generate",
			status:  http.StatusMethodNotAllowed,
			id:      "",
			rawUUID: "",
		},
	}

	for _, tt := range tests {
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
	handler := newTestHandler(t)
	codec, err := opid.NewCodec("usr")
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

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	codec, err := opid.NewCodec(resourcePrefix)
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(codec)
}
