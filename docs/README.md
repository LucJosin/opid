# Project guide

OPID gives UUIDv7 values a compact public form that includes a resource name and a checksum. This guide explains why the package exists, what it aims to provide, and how its format works. For a runnable introduction, see the [basic example](../examples/basic/main.go). The [project README](../README.md) shows the public API and local development commands.

## Project idea

A UUID identifies a resource, but its usual string form does not tell a reader what kind of resource it belongs to. OPID adds that context with a prefix such as `usr` for a user, `ws` for a workspace, or `dom` for a domain. The rest of the ID contains a Base62 representation of the UUID and a small checksum. Together, these parts give applications a consistent form for public URLs, responses, messages, and logs. Each project chooses its own prefixes without changing the encoding rules.

The caller can supply an existing UUIDv7, and OPID preserves its complete value. When a fresh ID is needed, `NewID` creates a UUIDv7 and returns its public form. There is no second UUID to create or save, and no mapping table is needed to recover the original. Encoding always gives the same public ID for the same UUID and prefix. Decoding returns the exact UUID that was supplied. This makes the package useful as a conversion step at an application boundary while leaving storage and resource handling to the application.

The checksum helps catch accidental changes during typing or copying. It covers the prefix as well as the encoded UUID, so changing a resource name also changes the expected checksum. A checksum is useful for detecting mistakes before an application uses an ID, but it cannot detect every possible change. Its limits are part of the format and should be understood by callers.

## Objectives

OPID aims to make public resource references compact without losing UUID data. It uses letters, digits, and underscores, which work in a URL path without extra escaping. The payload has no leading padding, so its length follows the UUID value. A prefix with three letters gives a public ID of at most 30 characters; shorter examples do not mean every UUID fits into the same payload length.

The main requirements are:

- Preserve every UUID bit so decoding an encoded value returns the original UUID.
- Accept one spelling per UUID and prefix so reencoding an accepted ID returns the same string.
- Reject malformed parts, extra padding, and values outside 128 bits.
- Enforce resource prefixes so a usr codec rejects a ws ID.
- Provide clear errors that callers can check without parsing error messages.
- Use Go 1.27's built in `uuid` package with no external dependencies.
- Allow one codec to be shared safely between goroutines.

## Non goals

OPID is a public representation, not a secret or cryptographic identifier. Anyone who knows the format can recover the UUID and read its UUIDv7 time information. The word opaque does not mean encrypted or hidden. A caller can also change an ID and calculate a new checksum, so the application must treat access control as a separate concern.

The package does not provide:

- Encryption, timestamp hiding, or protection from linking resource references.
- Proof of who created an ID or protection against deliberate changes.
- Permission to access a resource or proof that the resource exists.
- Data storage, database queries, HTTP handlers, repositories, or service layers.
- Support for UUID versions other than version 7 with the RFC variant.
- A fixed total string length or a promise that text sorting follows creation time.

## Public format

An OPID has three parts separated by exactly two underscores. The prefix identifies the resource kind, the payload holds the UUID value, and the checksum helps detect accidental changes. The following example represents UUID `0199c4e2-7a3b-7c12-8e42-9f1234567890` and contains 29 characters in total.

```text
<prefix>_<base62>_<checksum>
usr_31GQdK9BRjzTx1t1gYPlQ_ccn
```

`New` checks the prefix and returns a codec bound to it. It rejects invalid names instead of trimming or changing them. Create a codec once for each resource kind and reuse it for encoding and decoding. `Prefix()` returns the configured name, or an empty string for a nil or zero value codec. Other codec operations return `ErrInvalidPrefix` for nil or zero value codecs.

Prefix rules:

- Use 1 to `MaxPrefixLength` ASCII bytes, currently 32.
- Start with a lowercase letter.
- Use only lowercase letters and digits.
- Do not include spaces, uppercase letters, or underscores.

The payload treats the UUID's 16 bytes as one unsigned number, with the most significant byte first. Encoding works directly on that value without making a decimal string. Decoding restores all 16 bytes, including any leading zero bytes.

Payload rules:

- Use the alphabet `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`.
- Use at most 22 characters and omit leading zero padding.
- Preserve letter case because uppercase and lowercase characters have different values.
- Preserve the checksum's letter case too.

### Checksum

The checksum covers the ASCII bytes of the prefix, one underscore, and the payload. It uses the following rules so another implementation can produce the same result:

- Calculate CRC32 IEEE with reflected polynomial `0xedb88320`.
- Use initial value `0xffffffff` and final XOR `0xffffffff`.
- Reduce the result modulo 238328, which is 62 cubed.
- Encode that number with the payload's Base62 alphabet, most significant digit first.
- Use exactly three characters and keep any leading zeros.

The checksum input `123456789` produces `RME`. Unlike the payload, the checksum always has the same width.

There are 238328 possible checksum values. That is a deliberate space tradeoff for a short public ID, and it means collisions are possible. The checksum is intended to catch accidental mistakes, not deliberate changes. Including the prefix helps catch a resource name being changed while the payload and checksum are left untouched, but it does not stop someone from calculating a checksum for the changed name.

## ID type and codec

`opid.ID` is a named string type, so struct fields and function arguments can distinguish a public ID from ordinary text. Its zero value is an empty ID, and JSON represents it as a string without extra methods. The codec provides the following operations:

- `NewID()` creates a UUIDv7 with `uuid.NewV7` and returns its public ID.
- `Encode(id)` converts a UUID supplied by the caller into an `opid.ID`.
- `EncodeString(value)` parses UUID text with `uuid.Parse` and applies the same UUIDv7 checks as `Encode`.
- `Decode(id)` validates an `opid.ID` and returns its UUIDv7.
- `DecodeString(value)` applies the same validation to an incoming string.
- `Prefix()` returns the codec's configured resource name.

The ID type offers two small helpers:

- `String()` returns a plain string.
- `IsZero()` reports whether the ID is empty.

If the application needs the generated UUID for storage, it can decode the new ID or create the UUID first and pass it to `Encode`. `EncodeString` accepts the text forms supported by `uuid.Parse`, always produces the canonical public ID, and returns `ErrInvalidUUID` for invalid UUID text.

The type communicates intent rather than proving validity. A caller can convert any string to `ID`, and JSON can also supply invalid text. Neither step checks the prefix, checksum, or UUID. Validation stays in `Codec.Decode`, which knows the expected resource prefix. The ID itself has no prefix independent decode or validity method, so applications use the configured codec for those checks.

## Validation and errors

`Decode` validates input in a defined order. This gives predictable errors when an input has more than one problem and checks the checksum before the payload is used as a UUID:

1. Check the total length and the three part structure, including empty parts and extra separators.
2. Check the prefix against the codec and verify the payload characters.
3. Check the checksum before converting the payload into a number.
4. Reject redundant leading padding and numeric overflow.
5. Require UUID version 7 and RFC variant bits `10`.

Use `errors.Is` with the exported errors:

- `ErrInvalidFormat` for structure and length.
- `ErrInvalidPrefix` for an invalid or unexpected prefix.
- `ErrInvalidBase62` for invalid payload characters, leading padding, and overflow.
- `ErrInvalidChecksum` for checksum length or value.
- `ErrInvalidUUID` for unsupported UUID bits or invalid UUID text passed to `EncodeString`.

A failed decode returns a zero UUID along with the error. Applications choose how to present these failures; the package does not define HTTP status codes or an application error system.

## Implementation and performance

Base62 arithmetic uses two 64 bit words to hold the UUID's 128 bit value. Encoding repeatedly divides that value by 62 and uses each remainder to select a character. Decoding multiplies the current value by 62 and adds the next digit, checking carries so a number larger than 128 bits is rejected. The implementation uses Go's `math/bits` helpers for these operations and reads and writes the UUID bytes in the same order. This avoids processing all 16 bytes separately for every digit while preserving the format.

Encoding writes the prefix, payload, and checksum into a small local buffer before returning the final string. Decoding uses the checksum body already present in the input rather than constructing a new one. On the current Go 1.27.2 benchmarks, public encoding uses one allocation for the returned string and decoding uses none. Timing and allocation results should be checked again if the compiler or implementation changes. Codecs hold only their prefix and have no changing state, so encoding and decoding do not need locks or shared caches.

## Testing

The tests check the format through both the public API and private numeric helpers. Public API tests use `package opid_test` and build malformed inputs independently of the implementation. Private encoding tests compare Base62 results with large integer arithmetic and compare the checksum loop with `crc32.ChecksumIEEE`.

Coverage includes:

- Known UUID and checksum examples.
- Every bit boundary, including the boundary between the two 64 bit words and the largest 128 bit value.
- UUID version and variant bits, prefix limits, and damaged strings.
- Exact round trips and distinct UUID values.
- Concurrent conversion and `NewID` calls using a shared codec.
- Race checks for shared state problems.

Five fuzz targets explore different paths:

- `FuzzRoundTrip` tries UUID bytes and resource prefixes.
- `FuzzDecode` tries arbitrary public strings.
- `FuzzPayload` supplies correctly calculated checksums to reach numeric and UUID validation.
- `FuzzBase62` checks arbitrary 16 byte values against independent large integer arithmetic without forcing UUID version bits.
- `FuzzChecksum` compares arbitrary inputs with the standard library CRC32 function.

Serial and parallel benchmarks measure public encoding and decoding, while helper benchmarks measure Base62 and checksum work. The Makefile runs local checks with dependency downloads disabled. Use `make help` to see the commands or read the [development instructions](../README.md#development-and-examples).

Fuzz runs have two scopes:

- Normal CI uses five seconds per target to keep feedback quick.
- A weekly workflow gives each target five minutes and saves failing inputs. It can also be started manually.

For a longer local run, use `make fuzz FUZZ_TIME=5m`.
