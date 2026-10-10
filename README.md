# OpID

OpID is a Go package for public identifiers based on UUIDv7. It turns a UUID into a shorter string with a resource prefix and a checksum, and can recover the exact original UUID.

Use UUIDs inside your application and OpIDs in public URLs, responses, and messages. Requires Go 1.27 and uses only the standard library.

> [!WARNING]
> OpID is reversible, so it does not hide the UUID or its UUIDv7 time information. The checksum does not provide authentication or access control. Only UUID version 7 is supported.

## Use case

A user already has a UUIDv7. Your application can expose it with a name that tells the reader what it refers to:

```text
UUID:   0199c4e2-7a3b-7c12-8e42-9f1234567890
OpID:   usr_31GQdK9BRjzTx1t1gYPlQ_ccn
Format: <prefix>_<base62>_<checksum>
```

Create a codec for the resource, encode its UUID for public use, and decode incoming IDs with the same codec. Decoding checks the prefix, checksum, and UUID before returning the original value. The package needs no database or separate public UUID.

## Features

- Exact UUIDv7 round trips with no lost bits.
- Custom resource prefixes such as `usr`, `ws`, and `dom`.
- Short Base62 payloads with no leading padding.
- A three character checksum to catch accidental mistakes.
- Safe concurrent use with no external dependencies.

With a three letter prefix, an ID has at most 30 characters. Prefixes use lowercase letters and digits, start with a letter, and have at most 32 characters. Base62 is case sensitive.

## Example

Import `github.com/LucJosin/opid`:

```go
codec, err := opid.New("usr")
if err != nil {
    return err
}

publicID, err := codec.NewID()
if err != nil {
    return err
}

original, err := codec.Decode(publicID)
if err != nil {
    return err
}
```

`NewID` creates a fresh UUIDv7 and returns its public ID. Use `Encode` when you already have a UUID. Both return `opid.ID`, a named string type. `codec.Prefix()` returns its resource name. Use `String()` when a plain string is needed and `IsZero()` to check for an empty ID. For incoming strings, call `codec.DecodeString(value)`. Converting a string to `ID` does not validate it; the codec still checks the resource and format.

For a UUID received as text, use `codec.EncodeString(value)`. It parses the UUID and checks that it is version 7 before encoding.

For a complete runnable example:

```sh
go run ./examples/basic
```

## Error handling

Use `errors.Is` with `ErrInvalidFormat`, `ErrInvalidPrefix`, `ErrInvalidBase62`, `ErrInvalidChecksum`, or `ErrInvalidUUID`. Choose how to present these errors in your application. Failed decoding returns a zero UUID.

Read the [project guide](docs/README.md) for the ideas, objectives, non goals, and implementation details.

## Development and examples

The [basic example](examples/basic/main.go) shows encoding and decoding. The [HTTP example](examples/http/README.md) adds JSON endpoints to generate, encode, and decode IDs. Run it with `go run ./examples/http`. Use the Makefile for local checks:

```sh
make help
make fmt
make fmt-check
make lint
make vet
make test
make race
make fuzz
make bench
make cover
```

`make lint` runs the version pinned in `.golangci-lint-version` and downloads the tool when needed. Use `make help` for command descriptions and fuzz options. Licensed under Apache 2.0. See [LICENSE](LICENSE).
