// Package opid provides opaque IDs as short, reversible public UUIDv7 strings.
// It extends [shortuuid] encoding
// with resource prefixes, a checksum, and UUIDv7 checks.
// Clients should store and pass back the complete ID without reading its parts.
//
// IDs use the format prefix_payload_checksum. The payload has 22 Base57 characters,
// and the checksum has three Base62 characters. Letter case matters.
//
// Create a [Codec] with [NewCodec] and reuse it for one resource prefix. Use
// [New] for a new ID or [Codec.NewID] with a reused codec. Use [Codec.Encode]
// for an existing UUIDv7 and [Codec.Decode] to validate a public ID and
// recover its UUID. String inputs can use [Codec.EncodeString] and
// [Codec.DecodeString].
//
// Encoding does not hide the UUID or its timestamp. The checksum detects
// accidental changes, it does not provide authentication or access control.
//
// [shortuuid]: https://github.com/lithammer/shortuuid
package opid

import "strings"

// ID is a public identifier stored as a string. Its zero value is empty.
// Converting a string to ID does not validate it, use [Codec.Decode].
type ID string

// New creates a UUIDv7 and returns its public ID using prefix.
// It returns an empty ID and [ErrInvalidPrefix] if prefix is invalid.
//
// Use [NewCodec] to reuse a codec across calls.
func New(prefix string) (ID, error) {
	codec, err := NewCodec(prefix)
	if err != nil {
		return "", err
	}
	return codec.NewID()
}

// String returns the public identifier as a string.
func (id ID) String() string {
	return string(id)
}

// Prefix returns the resource prefix, or empty if it is missing or invalid.
// It does not validate the full ID, use [Codec.Decode] for that.
func (id ID) Prefix() string {
	prefix, _, ok := strings.Cut(id.String(), idSeparator)
	if !ok || !validPrefix(prefix) {
		return ""
	}

	return prefix
}

// IsZero reports whether id is empty.
func (id ID) IsZero() bool {
	return id == ""
}
