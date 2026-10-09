// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"encoding/binary"
	"github.com/gonz0w/horvium/internal/storage"
	"io"
)

const supervisorArgument = "--horvium-internal-property-supervisor"
const supervisorPayloadLimit = 64 << 10

func writeSupervisorRequest(w io.Writer, token [16]byte, boot bootIdentity, data []byte) error {
	if len(data) == 0 || len(data) > supervisorPayloadLimit {
		return ErrLimit
	}
	var header [36]byte
	copy(header[:16], token[:])
	copy(header[16:32], boot[:])
	binary.BigEndian.PutUint32(header[32:], uint32(len(data)))
	if n, err := w.Write(header[:]); err != nil {
		return err
	} else if n != len(header) {
		return io.ErrShortWrite
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}
func readSupervisorRequest(r io.Reader) ([16]byte, bootIdentity, storage.PropertyIntent, error) {
	var header [36]byte
	var token [16]byte
	var boot bootIdentity
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return token, boot, storage.PropertyIntent{}, ErrUnavailable
	}
	n := binary.BigEndian.Uint32(header[32:])
	if n == 0 || n > supervisorPayloadLimit {
		return token, boot, storage.PropertyIntent{}, ErrLimit
	}
	data := make([]byte, int(n))
	if _, err := io.ReadFull(r, data); err != nil {
		return token, boot, storage.PropertyIntent{}, ErrUnavailable
	}
	i, err := storage.DecodePropertyIntent(data)
	if err != nil {
		return token, boot, storage.PropertyIntent{}, err
	}
	if _, err = propertySetArguments(i); err != nil {
		return token, boot, storage.PropertyIntent{}, err
	}
	canonical, err := storage.EncodePropertyIntent(i)
	if err != nil || !bytes.Equal(data, canonical) {
		return token, boot, storage.PropertyIntent{}, storage.ErrInvalid
	}
	copy(token[:], header[:16])
	copy(boot[:], header[16:32])
	return token, boot, i, nil
}
func supervisorFrame(kind byte, token [16]byte, boot bootIdentity) [33]byte {
	var f [33]byte
	f[0] = kind
	copy(f[1:17], token[:])
	copy(f[17:], boot[:])
	return f
}
func readSupervisorFrame(r io.Reader, kind byte, token [16]byte, boot bootIdentity) error {
	var f [33]byte
	if _, err := io.ReadFull(r, f[:]); err != nil || f != supervisorFrame(kind, token, boot) {
		return ErrUnavailable
	}
	return nil
}
