// Package sri implements Subresource-Integrity-style content hashes
// (sha256/sha384/sha512, base64-encoded), used by direnv to fingerprint the
// contents an .envrc is allowed to load.
//
// Adapted from direnv v2.37.1 pkg/sri (MIT, (c) 2019 zimbatm and contributors).
// Pure; differentially tested against the pinned upstream package.
package sri

import (
	"crypto/sha256"
	"crypto/sha512"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"
)

// Algo is a supported hash algorithm — a closed set expressed as a Go+ enum so
// every consumer must handle all three cases.
type Algo enum {
	SHA256
	SHA384
	SHA512
}

// name is the SRI wire token for the algorithm ("sha256"/"sha384"/"sha512").
func (a Algo) name() string {
	match a {
	case SHA256:
		return "sha256"
	case SHA384:
		return "sha384"
	case SHA512:
		return "sha512"
	}
}

var b64Enc = b64.StdEncoding

// Hash is a parsed or computed SRI hash: an algorithm plus its raw digest.
type Hash struct {
	algo string
	sum  []byte
}

// String renders the hash in `<algo>-<base64>` form.
func (h *Hash) String() string {
	return h.algo + "-" + b64Enc.EncodeToString(h.sum)
}

// Hex renders the raw digest as a hex string.
func (h *Hash) Hex() string {
	return hex.EncodeToString(h.sum)
}

// Parse reads an `<algo>-<base64>` SRI string back into a Hash.
func Parse(sriHash string) (*Hash, error) {
	elems := strings.SplitN(sriHash, "-", 2)
	if len(elems) != 2 {
		return nil, fmt.Errorf("sri: not a hash %v", sriHash)
	}

	var algo Algo
	switch elems[0] {
	case "sha256":
		algo = SHA256
	case "sha384":
		algo = SHA384
	case "sha512":
		algo = SHA512
	default:
		return nil, fmt.Errorf("sri: unsupported algo %s", elems[0])
	}

	dbuf := make([]byte, b64Enc.DecodedLen(len(elems[1])))
	n, err := b64Enc.Decode(dbuf, []byte(elems[1]))
	if err != nil {
		return nil, err
	}
	sum := dbuf[:n]

	return &Hash{algo.name(), sum}, nil
}

// Writer tees everything written to it into an underlying writer while
// accumulating the SRI digest, which Sum then finalizes.
type Writer struct {
	w    io.Writer
	algo Algo
	h    hash.Hash
}

// NewWriter builds a Writer for the given algorithm. It panics on an
// unsupported algorithm, matching upstream.
func NewWriter(w io.Writer, algo Algo) Writer {
	var h hash.Hash
	match algo {
	case SHA256:
		h = sha256.New()
	case SHA384:
		h = sha512.New384()
	case SHA512:
		h = sha512.New()
	}
	return Writer{w, algo, h}
}

// Write forwards b to the underlying writer and, on success, into the digest.
func (w Writer) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	if err == nil {
		_, _ = w.h.Write(b)
	}
	return n, err
}

// Sum finalizes and returns the accumulated hash.
func (w Writer) Sum() *Hash {
	sum := w.h.Sum(nil)
	return &Hash{w.algo.name(), sum}
}
