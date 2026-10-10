// Package opid encodes UUIDv7 values as reversible, checksum-protected public IDs.
package opid

// ID is a public identifier. Use Codec.Decode to validate it.
type ID string

// String returns the public identifier as a string.
func (id ID) String() string {
	return string(id)
}

// IsZero reports whether id is empty.
func (id ID) IsZero() bool {
	return id == ""
}
