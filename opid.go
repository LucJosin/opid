// Package opid encodes UUIDv7 values as short, reversible public IDs.
//
// IDs use the format prefix_payload_checksum. The payload is unpadded Base62,
// and the checksum has three Base62 characters. Letter case matters.
//
// Create a [Codec] with [New] and reuse it for one resource prefix. Use
// [Codec.NewID] for a new ID, [Codec.Encode] for an existing UUIDv7, and
// [Codec.Decode] to validate a public ID and recover its UUID. String inputs
// can use [Codec.EncodeString] and [Codec.DecodeString].
//
// Encoding does not hide the UUID or its timestamp. The checksum detects
// accidental changes; it does not provide authentication or access control.
package opid

// ID is a public identifier stored as a string. Its zero value is empty.
// Converting a string to ID does not validate it, use [Codec.Decode].
type ID string

// String returns the public identifier as a string.
func (id ID) String() string {
	return string(id)
}

// IsZero reports whether id is empty.
func (id ID) IsZero() bool {
	return id == ""
}
