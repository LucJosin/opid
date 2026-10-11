# Project guide

OpID stands for opaque ID and gives UUIDv7 values a compact public form that includes a resource name and a checksum. This guide explains why the package exists, what it aims to provide, and how its format works. For a runnable introduction, see the [basic example](../examples/basic/main.go). The [project README](../README.md) shows the public API and local development commands.

## Project idea

OpID extends [shortuuid](https://github.com/lithammer/shortuuid) by adding a public ID format around its Base57 encoding. [Shortuuid](https://github.com/lithammer/shortuuid) handles the UUID payload. OpID adds the resource prefix and checksum, requires a fixed 22 character payload, and accepts only UUIDv7 with the RFC variant. It uses [shortuuid](https://github.com/lithammer/shortuuid) as a dependency without changing that library.

An opaque ID is a value that clients store and pass back as a complete string. Clients should not read its parts to decide how the application works. The application uses a codec to validate the string and recover the UUID. This keeps the public reference separate from the way the application handles its resources.

A UUID identifies a resource, but its usual string form does not tell a reader what kind of resource it belongs to. OpID adds that context with a prefix such as `usr` for a user, `ws` for a workspace, or `dom` for a domain. The rest of the ID contains a Base57 representation of the UUID and a small checksum. Together, these parts give applications a consistent form for public URLs, responses, messages, and logs. Each project chooses its own prefixes without changing the encoding rules.

The caller can supply an existing UUIDv7, and OpID preserves its complete value. When a fresh ID is needed, `NewID` creates a UUIDv7 and returns its public form. There is no second UUID to create or save, and no mapping table is needed to recover the original. Encoding always gives the same public ID for the same UUID and prefix. Decoding returns the exact UUID that was supplied. This makes the package useful as a conversion step at an application boundary while leaving storage and resource handling to the application.

The checksum helps catch accidental changes during typing or copying. It covers the prefix as well as the encoded UUID, so changing a resource name also changes the expected checksum. A checksum is useful for detecting mistakes before an application uses an ID, but it cannot detect every possible change. Its limits are part of the format and should be understood by callers.

## Objectives

OpID aims to make public resource references compact without losing UUID data. It uses letters, digits, and underscores, which work in a URL path without extra escaping. The payload always has 22 characters, including leading padding when needed. A prefix with three letters gives a public ID of exactly 30 characters. The total length stays fixed for each prefix and equals the prefix length plus 27 characters.

The main requirements are:

- Preserve every UUID bit so decoding an encoded value returns the original UUID.
- Accept one spelling per UUID and prefix so reencoding an accepted ID returns the same string.
- Reject malformed parts, extra padding, and values outside 128 bits.
- Enforce resource prefixes so a usr codec rejects a ws ID.
- Provide clear errors that callers can check without parsing error messages.
- Use Go 1.27's built in `uuid` package and [shortuuid](https://github.com/lithammer/shortuuid) for payload encoding.
- Allow one codec to be shared safely between goroutines.

## Non goals

OpID is a public representation, not a secret or cryptographic identifier. Anyone who knows the format can recover the UUID and read its UUIDv7 time information. Here, opaque describes how clients should use the ID. The encoding is public and reversible, so it does not hide the UUID. A caller can also change an ID and calculate a new checksum, so the application must treat access control as a separate concern.

The package does not provide:

- Encryption, timestamp hiding, or protection from linking resource references.
- Proof of who created an ID or protection against deliberate changes.
- Permission to access a resource or proof that the resource exists.
- Data storage, database queries, HTTP handlers, repositories, or service layers.
- Support for UUID versions other than version 7 with the RFC variant.
- One total string length across different prefix lengths or a promise that text sorting follows creation time.

## Public format

An OpID has three parts separated by exactly two underscores. The prefix identifies the resource kind, the payload holds the UUID value, and the checksum helps detect accidental changes. The following example represents UUID `0199c4e2-7a3b-7c12-8e42-9f1234567890` and contains 30 characters in total.

```text
<prefix>_<base57>_<checksum>
usr_2JFJNM9BCXm7nJtPJdJuB5_lZw
```

`NewCodec` checks the prefix and returns a codec bound to it. It rejects invalid names instead of trimming or changing them. Create a codec once for each resource kind and reuse it for encoding and decoding. `Prefix()` returns the configured name, or an empty string for a nil or zero value codec. Other codec operations return `ErrInvalidPrefix` for nil or zero value codecs.

Prefix rules:

- Use 1 to `MaxPrefixLength` ASCII bytes, currently 32.
- Start with a lowercase letter.
- Use only lowercase letters and digits.
- Do not include spaces, uppercase letters, or underscores.

The payload treats the UUID's 16 bytes as one unsigned number, with the most significant byte first. Encoding works directly on that value without making a decimal string. Decoding restores all 16 bytes, including any leading zero bytes.

Payload rules:

- Use the alphabet `23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz`.
- Avoid the similar looking characters `0`, `O`, `1`, `I`, and `l`.
- Use exactly 22 characters, including any leading `2` padding.
- Reject shorter and longer payloads, even when their checksum is valid.
- Preserve letter case because uppercase and lowercase characters have different values.
- Preserve the checksum's letter case too.

### Checksum

The checksum covers the ASCII bytes of the prefix, one underscore, and the payload. It uses the following rules so another implementation can produce the same result:

- Calculate CRC32 IEEE with reflected polynomial `0xedb88320`.
- Use initial value `0xffffffff` and final XOR `0xffffffff`.
- Reduce the result modulo 238328, which is 62 cubed.
- Encode that number with `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`, most significant digit first.
- Use exactly three characters and keep any leading zeros.

The checksum input `123456789` produces `RME`. The checksum uses Base62 while the payload uses Base57, so similar looking characters may still appear in the checksum. Both parts have a fixed width.

There are 238328 possible checksum values. That is a deliberate space tradeoff for a short public ID, and it means collisions are possible. The checksum is intended to catch accidental mistakes, not deliberate changes. Including the prefix helps catch a resource name being changed while the payload and checksum are left untouched, but it does not stop someone from calculating a checksum for the changed name.

## ID type and codec

`opid.ID` is a named string type, so struct fields and function arguments can distinguish a public ID from ordinary text. Its zero value is an empty ID, and JSON represents it as a string without extra methods. The codec provides the following operations:

- `New(prefix)` creates a UUIDv7 and returns its public ID.
- `NewCodec(prefix)` validates the prefix and returns a reusable codec.
- `NewID()` creates a UUIDv7 with `uuid.NewV7` and returns its public ID.
- `Encode(id)` converts a UUID supplied by the caller into an `opid.ID`.
- `EncodeString(value)` parses UUID text with `uuid.Parse` and applies the same UUIDv7 checks as `Encode`.
- `Decode(id)` validates an `opid.ID` and returns its UUIDv7.
- `DecodeString(value)` applies the same validation to an incoming string.
- `Prefix()` returns the codec's configured resource name.

The ID type offers three small helpers:

- `String()` returns a plain string.
- `Prefix()` returns the text before the first underscore when it follows the prefix rules, or empty when it is missing or invalid. It does not check the payload or checksum.
- `IsZero()` reports whether the ID is empty.

If the application needs the generated UUID for storage, it can decode the new ID or create the UUID first and pass it to `Encode`. `EncodeString` accepts the text forms supported by `uuid.Parse`, always produces the canonical public ID, and returns `ErrInvalidUUID` for invalid UUID text.

The type communicates intent rather than proving validity. A caller can convert any string to `ID`, and JSON can also supply invalid text. Neither step checks the prefix, checksum, or UUID. Validation stays in `Codec.Decode`, which knows the expected resource prefix. The ID itself has no prefix independent decode or validity method, so applications use the configured codec for those checks.

## Validation and errors

`Decode` validates input in a defined order. This gives predictable errors when an input has more than one problem and checks the checksum before the payload is used as a UUID:

1. Check the total length and the three part structure, including empty parts and extra separators.
2. Check the prefix against the codec, require a 22 character payload, and verify its characters.
3. Check the checksum before converting the payload into a number.
4. Decode with [shortuuid](https://github.com/lithammer/shortuuid) and reject numeric overflow.
5. Require UUID version 7 and RFC variant bits `10`.

Use `errors.Is` with the exported errors:

- `ErrInvalidFormat` for structure and length.
- `ErrInvalidPrefix` for an invalid or unexpected prefix.
- `ErrInvalidPayload` for invalid Base57 payload characters and overflow.
- `ErrInvalidChecksum` for checksum length or value.
- `ErrInvalidUUID` for unsupported UUID bits or invalid UUID text passed to `EncodeString`.

A failed decode returns a zero UUID along with the error. Applications choose how to present these failures; the package does not define HTTP status codes or an application error system.

## Implementation and performance

Payload encoding and decoding use [shortuuid v5](https://github.com/lithammer/shortuuid), pinned in go.mod. It supports the standard library UUID type and has no transitive dependencies. The package creates one encoder for the default Base57 alphabet and reuses it. Opid keeps the resource prefix, checksum, UUID version, and UUID variant checks around this encoder.

[Shortuuid](https://github.com/lithammer/shortuuid) accepts shorter strings during decoding by filling in missing leading digits. Opid rejects those strings before decoding because its payload must contain exactly 22 characters. Valid alphabet characters, fixed width, and the 128 bit range together give one spelling per UUID. Changing or removing a payload character still requires a valid checksum and supported UUID bits before the ID is accepted.

Encoding writes the prefix, payload, and checksum into a small local buffer before returning the final string. Decoding uses the checksum body already present in the input rather than constructing a new one. On the local Go 1.27.2 benchmarks, public encoding uses two allocations, one for the payload and one for the returned string. Decoding uses none. Results should be checked again if the compiler or implementation changes. Codecs hold only their prefix, and the shared encoder has no changing state, so calls can run safely across goroutines.

## Testing

The tests check the format through both the public API and private numeric helpers. Public API tests use `package opid_test` and build malformed inputs independently of the implementation. Private encoding tests compare Base57 results with large integer arithmetic and compare the checksum loop with `crc32.ChecksumIEEE`.

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
- `FuzzBase57` checks arbitrary 16 byte values against independent large integer arithmetic without forcing UUID version bits.
- `FuzzChecksum` compares arbitrary inputs with the standard library CRC32 function.

Serial and parallel benchmarks measure public encoding and decoding, while helper benchmarks measure Base57 and checksum work. The Makefile provides formatting, lint, tests, and benchmarks. Dependency downloads follow the caller's Go settings. Use `make help` to see the commands or read the [development instructions](../README.md#development-and-examples).

Fuzz runs have two scopes:

- Normal CI uses five seconds per target to keep feedback quick.
- A weekly workflow gives each target five minutes and saves failing inputs. It can also be started manually.

For a longer local run, use `make fuzz FUZZ_TIME=5m`.
