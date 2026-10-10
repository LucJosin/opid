# HTTP example

This example runs a small JSON server using Go's net/http package. It shows how an application can generate a UUIDv7 and expose it as an OPID, or convert between an existing UUIDv7 and its public ID. It uses GET endpoints with path parameters and needs no database or request body.

## Run

Start the server from the project root:

```sh
go run ./examples/http
```

The server listens on 127.0.0.1:8080. The application creates one usr codec at startup and shares it across all three handlers. Codecs are safe for concurrent use, so requests do not need to create a new one. Routes use the configured resource name, and other resource paths return status 404. Use a separate terminal for the requests below.

## Generate

Generate a fresh UUIDv7 and receive both its raw value and public ID:

```sh
curl http://127.0.0.1:8080/usr/generate
```

The response has status 200 with `uuid` and `id` fields. Their values change on each request; this is one possible response:

```json
{"id":"usr_31GQdK9BRjzTx1t1gYPlQ_ccn","uuid":"0199c4e2-7a3b-7c12-8e42-9f1234567890"}
```

The handler creates a UUIDv7 with `uuid.NewV7` and encodes it with the shared codec, keeping both values available. Use `codec.NewID()` when only the public ID is needed. Responses use Cache Control with no store, and no resource is saved.

## Encode

Pass an existing UUIDv7 in the path:

```sh
curl http://127.0.0.1:8080/usr/encode/0199c4e2-7a3b-7c12-8e42-9f1234567890
```

The response has status 200:

```json
{"id":"usr_31GQdK9BRjzTx1t1gYPlQ_ccn"}
```

## Decode

Pass a public ID in the path to recover its UUIDv7:

```sh
curl http://127.0.0.1:8080/usr/decode/usr_31GQdK9BRjzTx1t1gYPlQ_ccn
```

The response has status 200:

```json
{"uuid":"0199c4e2-7a3b-7c12-8e42-9f1234567890"}
```

The generate and decode endpoints return raw UUIDs to demonstrate the relationship between the two forms. In an application, this step can instead pass the UUID to internal code and return only the public resource data.

## Errors and scope

The example handles failures as follows:

- Unsupported UUIDs and broken IDs return status 400 with a JSON error field.
- A ws ID sent to the usr decode route is rejected.
- Missing path values and unsupported resource paths return status 404.
- Unsupported methods return status 405.

The standard router provides the 404 and 405 responses.

This is a local demonstration, not an authenticated resource API. The application chooses its prefix, and no resource is created or looked up. For other resource kinds, create one codec per prefix at startup and reuse it in the matching handlers. Check access separately. The example tests use in memory HTTP requests and do not open a network connection.
