# httpcat

A minimal command-line HTTP client written in Go, supporting HTTP/1.1, HTTP/2, and HTTP/3.

## Install

```sh
go install github.com/aaronriekenberg/httpcat/cmd/httpcat@latest
```

Or build from source:

```sh
git clone https://github.com/aaronriekenberg/httpcat
cd httpcat
go build -o httpcat ./cmd/httpcat
```

## Usage

```
httpcat [options] <url>
```

Only `http://` and `https://` URLs are supported.

## Options

| Flag | Description |
|------|-------------|
| `--http2-prior-knowledge` | Use HTTP/2 without HTTP/1.1 Upgrade (RFC 7540 section 3.4) |
| `--http3` | Attempt HTTP/3 (QUIC), fall back to HTTP/1.x or HTTP/2 on failure |
| `--http3-only` | Require HTTP/3; exit with an error if unavailable |
| `-k`, `--insecure` | Skip TLS certificate verification |
| `-v`, `--verbose` | Print request and response headers to stderr |
| `-X`, `--request <method>` | HTTP method to use (default: `GET`) |
| `-h`, `--help` | Show help |

## Examples

Simple GET request:

```sh
httpcat https://example.com
```

Verbose output showing request and response headers:

```sh
httpcat -v https://httpbin.org/get
```

```
* Connecting to httpbin.org
> GET /get HTTP/1.1
> Host: httpbin.org
>
* Protocol: HTTP/2.0
< 200 OK
< Content-Type: application/json
< ...
<
{ ... }
```

POST request:

```sh
httpcat -X POST https://httpbin.org/post
```

Force HTTP/3 with fallback to HTTP/2 or HTTP/1.1:

```sh
httpcat --http3 -v https://cloudflare.com/
```

Require HTTP/3, fail if unavailable:

```sh
httpcat --http3-only https://cloudflare.com/
```

Force HTTP/2 with prior knowledge (no HTTP/1.1 upgrade roundtrip):

```sh
httpcat --http2-prior-knowledge https://example.com
```

Skip TLS certificate verification (e.g. for local development):

```sh
httpcat -k https://localhost:8443/
```

Bundle short flags:

```sh
httpcat -kv https://localhost:8443/
```

## HTTP version selection

| Condition | Protocol used |
|-----------|---------------|
| Default | HTTP/1.1 or HTTP/2 via ALPN negotiation (`net/http`) |
| `--http2-prior-knowledge` | HTTP/2 directly, no upgrade (`golang.org/x/net/http2`) |
| `--http3` | HTTP/3 via QUIC; falls back if server unreachable over QUIC |
| `--http3-only` | HTTP/3 via QUIC; exits non-zero on failure |

HTTP/3 is implemented using [quic-go](https://github.com/quic-go/quic-go).

## Extending

To add a new flag:

1. Add a `flag` entry to `flagDefs` in `internal/cli/options.go`.
2. Add the corresponding field to the `Options` struct.
3. Handle it in the `applyFlag` switch.
4. Read the field in the client layer (`internal/client/`).

No other files need to change.

## License

MIT
