// Package cli handles command-line argument parsing for httpcat.
// Add new flags here to extend the tool's capabilities.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Options holds all parsed command-line options.
type Options struct {
	// Protocol selection
	HTTP2PriorKnowledge bool // --http2-prior-knowledge
	HTTP3               bool // --http3: try HTTP/3, fall back to HTTP/1.x or HTTP/2
	HTTP3Only           bool // --http3-only: fail if HTTP/3 not available

	// TLS
	Insecure bool // -k, --insecure: skip TLS certificate verification

	// Output
	Verbose bool // -v, --verbose: print request/response headers to stderr

	// Request
	Method string // -X, --request: HTTP method (default: GET)

	// Meta
	Version bool // -V, --version: print version and exit

	// Positional
	URL string
}

// flag represents a single CLI flag definition.
type flag struct {
	long     string
	short    string
	takesArg bool
	helpText string
}

var flagDefs = []flag{
	{long: "http2-prior-knowledge", short: "", takesArg: false, helpText: "Use HTTP/2 without HTTP/1.1 Upgrade (prior knowledge)"},
	{long: "http3", short: "", takesArg: false, helpText: "Attempt HTTP/3, fall back on failure"},
	{long: "http3-only", short: "", takesArg: false, helpText: "Use HTTP/3 only, fail if not available"},
	{long: "insecure", short: "k", takesArg: false, helpText: "Skip TLS certificate verification"},
	{long: "verbose", short: "v", takesArg: false, helpText: "Print request and response headers to stderr"},
	{long: "request", short: "X", takesArg: true, helpText: "HTTP method to use (default: GET)"},
	{long: "version", short: "V", takesArg: false, helpText: "Print version information and exit"},
}

// Parse parses os.Args[1:] into Options.
func Parse(args []string) (*Options, error) {
	opts := &Options{
		Method: "GET",
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "--help" || arg == "-h" {
			printUsage()
			os.Exit(0)
		}

		if strings.HasPrefix(arg, "--") {
			// Long flag
			name := arg[2:]
			var val string
			if idx := strings.IndexByte(name, '='); idx >= 0 {
				val = name[idx+1:]
				name = name[:idx]
			}

			fd := findLongFlag(name)
			if fd == nil {
				return nil, fmt.Errorf("unknown option: --%s", name)
			}

			if fd.takesArg {
				if val == "" {
					i++
					if i >= len(args) {
						return nil, fmt.Errorf("option --%s requires an argument", name)
					}
					val = args[i]
				}
				if err := applyFlag(opts, fd.long, val); err != nil {
					return nil, err
				}
			} else {
				if err := applyFlag(opts, fd.long, ""); err != nil {
					return nil, err
				}
			}

		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flags (may be bundled, e.g. -kv)
			chars := arg[1:]
			for ci := 0; ci < len(chars); ci++ {
				ch := string(chars[ci])
				fd := findShortFlag(ch)
				if fd == nil {
					return nil, fmt.Errorf("unknown option: -%s", ch)
				}
				if fd.takesArg {
					// Remainder of this token is the value, or next token
					var val string
					if ci+1 < len(chars) {
						val = chars[ci+1:]
						ci = len(chars) // consume rest
					} else {
						i++
						if i >= len(args) {
							return nil, fmt.Errorf("option -%s requires an argument", ch)
						}
						val = args[i]
					}
					if err := applyFlag(opts, fd.long, val); err != nil {
						return nil, err
					}
				} else {
					if err := applyFlag(opts, fd.long, ""); err != nil {
						return nil, err
					}
				}
			}

		} else {
			// Positional argument: the URL
			if opts.URL != "" {
				return nil, errors.New("multiple URLs provided; only one is supported")
			}
			opts.URL = arg
		}

		i++
	}

	if opts.URL == "" {
		if opts.Version {
			return opts, nil
		}
		return nil, errors.New("no URL provided")
	}

	if opts.HTTP3 && opts.HTTP2PriorKnowledge {
		return nil, errors.New("--http3 and --http2-prior-knowledge are mutually exclusive")
	}
	if opts.HTTP3Only && opts.HTTP2PriorKnowledge {
		return nil, errors.New("--http3-only and --http2-prior-knowledge are mutually exclusive")
	}
	if opts.HTTP3Only {
		opts.HTTP3 = true
	}

	if !opts.Version && !strings.HasPrefix(opts.URL, "http://") && !strings.HasPrefix(opts.URL, "https://") {
		return nil, fmt.Errorf("unsupported scheme in URL %q (only http and https are supported)", opts.URL)
	}

	return opts, nil
}

func applyFlag(opts *Options, long, val string) error {
	switch long {
	case "http2-prior-knowledge":
		opts.HTTP2PriorKnowledge = true
	case "http3":
		opts.HTTP3 = true
	case "http3-only":
		opts.HTTP3Only = true
	case "insecure":
		opts.Insecure = true
	case "verbose":
		opts.Verbose = true
	case "request":
		opts.Method = strings.ToUpper(val)
	case "version":
		opts.Version = true
	default:
		return fmt.Errorf("unhandled flag: %s", long)
	}
	return nil
}

func findLongFlag(name string) *flag {
	for i := range flagDefs {
		if flagDefs[i].long == name {
			return &flagDefs[i]
		}
	}
	return nil
}

func findShortFlag(ch string) *flag {
	for i := range flagDefs {
		if flagDefs[i].short == ch {
			return &flagDefs[i]
		}
	}
	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: httpcat [options] <url>

httpcat is a minimal HTTP client supporting HTTP/1.1, HTTP/2 and HTTP/3.
Only http:// and https:// URLs are supported.

Options:
`)
	for _, fd := range flagDefs {
		if fd.short != "" {
			fmt.Fprintf(os.Stderr, "  -%s, --%-30s %s\n", fd.short, fd.long, fd.helpText)
		} else {
			fmt.Fprintf(os.Stderr, "      --%-30s %s\n", fd.long, fd.helpText)
		}
	}
	fmt.Fprintf(os.Stderr, "  -h, --%-30s Show this help\n", "help")
}
