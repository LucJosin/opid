package opid

import (
	"errors"
	"strings"
	"uuid"
)

var (
	// ErrInvalidFormat indicates an invalid OPID structure or length.
	ErrInvalidFormat = errors.New("opid: invalid format")
	// ErrInvalidPrefix indicates an invalid or unexpected resource prefix.
	ErrInvalidPrefix = errors.New("opid: invalid prefix")
	// ErrInvalidChecksum indicates a malformed or mismatched checksum.
	ErrInvalidChecksum = errors.New("opid: invalid checksum")
	// ErrInvalidBase62 indicates invalid payload characters, padding, or overflow.
	ErrInvalidBase62 = errors.New("opid: invalid base62")
	// ErrInvalidUUID indicates malformed UUID text or an unsupported UUID value.
	ErrInvalidUUID = errors.New("opid: invalid UUIDv7")
)

// Codec encodes and decodes IDs for one prefix. It is safe for concurrent use.
type Codec struct {
	prefix string
}

// MaxPrefixLength is the maximum prefix length in bytes.
const MaxPrefixLength = 32

const (
	minPrefixLength  = 1
	minPayloadLength = 1
	separatorCount   = 2
	idSeparator      = "_"

	minIDLength = minPrefixLength + separatorCount + minPayloadLength + checksumSize
	maxIDLength = MaxPrefixLength + separatorCount + payloadWidth + checksumSize

	// UUID version and variant fields defined by RFC 9562.
	uuidVersionByte    = 6
	uuidVersionMask    = 0xf0
	uuidV7Bits         = 0x70
	uuidVariantByte    = 8
	uuidVariantMask    = 0xc0
	uuidRFCVariantBits = 0x80
)

// New returns a codec bound to prefix.
//
// The prefix must contain 1 to [MaxPrefixLength] ASCII bytes, start with a
// lowercase letter, and contain only lowercase letters and digits. Prefixes
// such as "usr", "ws", and "dom" are valid. Underscores, uppercase letters,
// spaces, and empty prefixes are rejected with [ErrInvalidPrefix]. New does
// not trim or change the prefix.
//
// The codec uses this prefix when encoding and requires it when decoding.
// It is safe for concurrent use. Create a separate codec for each resource
// kind; no prefix registry is needed.
//
// For example:
//
//	codec, err := opid.New("usr")
//	if err != nil {
//	    return err
//	}
//	publicID, err := codec.EncodeString("0199c4e2-7a3b-7c12-8e42-9f1234567890")
func New(prefix string) (*Codec, error) {
	if !validPrefix(prefix) {
		return nil, ErrInvalidPrefix
	}

	return &Codec{prefix: prefix}, nil
}

// Prefix returns the resource prefix, or empty for a nil or zero value codec.
func (c *Codec) Prefix() string {
	if c == nil {
		return ""
	}

	return c.prefix
}

// NewID creates a UUIDv7 and returns its public ID using the codec's prefix.
// It returns an empty ID and [ErrInvalidPrefix] for a nil or zero value codec.
func (c *Codec) NewID() (ID, error) {
	if c == nil || c.prefix == "" {
		return "", ErrInvalidPrefix
	}

	return c.Encode(uuid.NewV7())
}

// EncodeString parses s as UUIDv7 and returns its canonical public ID.
func (c *Codec) EncodeString(s string) (ID, error) {
	if c == nil || c.prefix == "" {
		return "", ErrInvalidPrefix
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return "", ErrInvalidUUID
	}

	return c.Encode(id)
}

// Encode returns the canonical public ID for id using the codec's prefix.
func (c *Codec) Encode(id uuid.UUID) (ID, error) {
	if c == nil || c.prefix == "" {
		return "", ErrInvalidPrefix
	}

	if !validUUID(id) {
		return "", ErrInvalidUUID
	}

	var payload [payloadWidth]byte
	encoded := encodePayload(&payload, id)

	var out [maxIDLength]byte
	n := copy(out[:], c.prefix)

	out[n] = idSeparator[0]
	n++
	n += copy(out[n:], encoded)

	sum := checksumBytes(out[:n])
	out[n] = idSeparator[0]
	n++
	n += copy(out[n:], sum[:])

	return ID(out[:n]), nil
}

// DecodeString validates s and returns its UUIDv7.
// It applies [Codec.Decode] with the same errors and validation rules.
func (c *Codec) DecodeString(s string) (uuid.UUID, error) {
	return c.Decode(ID(s))
}

// Decode validates publicID and returns its original UUIDv7.
func (c *Codec) Decode(publicID ID) (uuid.UUID, error) {
	if c == nil || c.prefix == "" {
		return uuid.UUID{}, ErrInvalidPrefix
	}

	if len(publicID) < minIDLength || len(publicID) > maxIDLength {
		return uuid.UUID{}, ErrInvalidFormat
	}

	prefix, rest, ok := strings.Cut(publicID.String(), idSeparator)
	if !ok {
		return uuid.UUID{}, ErrInvalidFormat
	}

	payload, sum, ok := strings.Cut(rest, idSeparator)
	if !ok || strings.Contains(sum, idSeparator) {
		return uuid.UUID{}, ErrInvalidFormat
	}

	if prefix == "" || payload == "" || sum == "" {
		return uuid.UUID{}, ErrInvalidFormat
	}

	if !validPrefix(prefix) || prefix != c.prefix {
		return uuid.UUID{}, ErrInvalidPrefix
	}

	if len(payload) > payloadWidth {
		return uuid.UUID{}, ErrInvalidFormat
	}

	for i := range payload {
		if digit62(payload[i]) < 0 {
			return uuid.UUID{}, ErrInvalidBase62
		}
	}

	bodyEnd := len(publicID) - len(sum) - len(idSeparator)
	body := publicID[:bodyEnd]
	expected := checksumBytes([]byte(body))
	if len(sum) != checksumSize || sum != string(expected[:]) {
		return uuid.UUID{}, ErrInvalidChecksum
	}

	id, err := decodeBase62(payload)
	if err != nil {
		return uuid.UUID{}, err
	}

	if !validUUID(id) {
		return uuid.UUID{}, ErrInvalidUUID
	}

	return id, nil
}

func validPrefix(prefix string) bool {
	if len(prefix) < minPrefixLength || len(prefix) > MaxPrefixLength {
		return false
	}

	if prefix[0] < 'a' || prefix[0] > 'z' {
		return false
	}

	for i := range prefix {
		b := prefix[i]
		isLetter := b >= 'a' && b <= 'z'
		isDigit := b >= '0' && b <= '9'
		if !isLetter && !isDigit {
			return false
		}
	}

	return true
}

func validUUID(id uuid.UUID) bool {
	version := id[uuidVersionByte] & uuidVersionMask
	variant := id[uuidVariantByte] & uuidVariantMask
	return version == uuidV7Bits && variant == uuidRFCVariantBits
}
