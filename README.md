# httpcat

A minimal command-line HTTP client written in Go, supporting HTTP/1.1, HTTP/2, and HTTP/3.

## Install

1. Download the latest release binary for linux/mac/windows from [github releases](https://github.com/aaronriekenberg/httpcat/releases).  Binaries are in Assets for every release.

2. go install:

```sh
go install github.com/aaronriekenberg/httpcat@latest
```

3. build from source:

```sh
git clone https://github.com/aaronriekenberg/httpcat
cd httpcat
go build -o httpcat .
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
| `-H`, `--header <value>` | Add a request header (e.g., `Content-Type: application/json`); can be used multiple times |
| `--json <value>` | Send JSON body; can be a direct string, `@-` to read from stdin, or `@filename` to read from file |
| `--data-binary <value>` | Send binary body; can be `@-` to read from stdin, or `@filename` to read from file |
| `-V`, `--version` | Print version information and exit |
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

Send custom headers:

```sh
httpcat -H "Content-Type: application/json" -H "Authorization: Bearer token" https://httpbin.org/post
```

Send JSON body as direct string:

```sh
httpcat --json '{"hello": "world"}' https://httpbin.org/post
```

Send JSON body from file:

```sh
httpcat --json @request.json -X POST https://httpbin.org/post
```

Send JSON body from stdin:

```sh
echo '{"key": "value"}' | httpcat --json @- -X POST https://httpbin.org/post
```

Send binary body from file:

```sh
httpcat --data-binary @image.png -X POST https://httpbin.org/post
```

Send binary body from stdin:

```sh
cat binary_data.bin | httpcat --data-binary @- -X POST https://httpbin.org/post
```

Print version:

```sh
httpcat --version
httpcat -V
```

```
httpcat v1.0.0 (commit abc1234, built 2026-07-03T12:00:00Z)
```

## HTTP version selection

| Condition | Protocol used |
|-----------|---------------|
| Default | HTTP/1.1 or HTTP/2 via ALPN negotiation (`net/http`) |
| `--http2-prior-knowledge` + `http://` | Unencrypted HTTP/2 (h2c) directly over TCP (`net/http`) |
| `--http2-prior-knowledge` + `https://` | HTTP/2 over TLS, no HTTP/1.1 upgrade (`net/http`) |
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
