// Package gzenv implements a compact environment format using json+zlib+base64.
// direnv uses it to round-trip a whole environment snapshot back into itself.
//
// Adapted from direnv v2.37.1 gzenv (MIT, (c) 2019 zimbatm and contributors).
// The wire format is byte-for-byte compatible with upstream; differentially
// tested against the pinned upstream package.
package gzenv

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
)

// Marshal encodes obj into the gzenv format. It panics if obj is not
// JSON-encodable, matching upstream.
func Marshal(obj any) string {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		panic(fmt.Errorf("marshal(): %w", err))
	}

	zlibData := bytes.NewBuffer([]byte{})
	w := zlib.NewWriter(zlibData)
	// The zlib writer over an in-memory buffer never fails on Write.
	_, _ = w.Write(jsonData)
	if err := w.Close(); err != nil {
		log.Printf("Warning: failed to close zlib writer: %v", err)
	}

	return base64.URLEncoding.EncodeToString(zlibData.Bytes())
}

// Unmarshal restores a gzenv string back into obj.
func Unmarshal(gzenv string, obj any) error {
	gzenv = strings.TrimSpace(gzenv)

	data, err := base64.URLEncoding.DecodeString(gzenv)
	if err != nil {
		return fmt.Errorf("unmarshal() base64 decoding: %w", err)
	}

	zlibReader := bytes.NewReader(data)
	w, err := zlib.NewReader(zlibReader)
	if err != nil {
		return fmt.Errorf("unmarshal() zlib opening: %w", err)
	}

	envData := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(envData, w); err != nil {
		return fmt.Errorf("unmarshal() zlib decoding: %w", err)
	}
	if err := w.Close(); err != nil {
		log.Printf("Warning: failed to close zlib reader: %v", err)
	}

	if err := json.Unmarshal(envData.Bytes(), &obj); err != nil {
		return fmt.Errorf("unmarshal() json parsing: %w", err)
	}

	return nil
}
