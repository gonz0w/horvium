// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"encoding/binary"
	"github.com/gonz0w/horvium/internal/storage"
	"testing"
)

func TestSupervisorRequestCanonicalAndBounded(t *testing.T) {
	_, i, _, _, _, _, _ := controlFixture(t)
	data, err := storage.EncodePropertyIntent(i)
	if err != nil {
		t.Fatal(err)
	}
	token := [16]byte{7}
	boot := bootIdentity{9}
	var buf bytes.Buffer
	if writeSupervisorRequest(&buf, token, boot, data) != nil {
		t.Fatal("valid typed request refused")
	}
	wire := append([]byte(nil), buf.Bytes()...)
	got, b, decoded, err := readSupervisorRequest(&buf)
	out, _ := storage.EncodePropertyIntent(decoded)
	if err != nil || got != token || b != boot || !bytes.Equal(out, data) {
		t.Fatal("lost original bounded request")
	}
	for _, bad := range [][]byte{nil, wire[:35], wire[:len(wire)-1], append(append([]byte(nil), wire[:36]...), []byte("invalid")...)} {
		if _, _, _, e := readSupervisorRequest(bytes.NewReader(bad)); e == nil {
			t.Fatal("incomplete or invalid typed request accepted")
		}
	}
	oversized := append([]byte(nil), wire...)
	binary.BigEndian.PutUint32(oversized[32:], supervisorPayloadLimit+1)
	if _, _, _, e := readSupervisorRequest(bytes.NewReader(oversized)); e == nil {
		t.Fatal("unchecked allocation length")
	}
	if writeSupervisorRequest(&buf, token, boot, nil) == nil {
		t.Fatal("empty payload accepted")
	}
}
func TestSupervisorFramesBindOriginalPhaseAndIdentity(t *testing.T) {
	token := [16]byte{1}
	boot := bootIdentity{2}
	frame := supervisorFrame(1, token, boot)
	if readSupervisorFrame(bytes.NewReader(frame[:]), 1, token, boot) != nil {
		t.Fatal("valid ready refused")
	}
	for _, kind := range []byte{0, 2, 3} {
		if readSupervisorFrame(bytes.NewReader(frame[:]), kind, token, boot) == nil {
			t.Fatal("phase replay accepted")
		}
	}
	if readSupervisorFrame(bytes.NewReader(frame[:32]), 1, token, boot) == nil || readSupervisorFrame(bytes.NewReader(frame[:]), 1, [16]byte{3}, boot) == nil || readSupervisorFrame(bytes.NewReader(frame[:]), 1, token, bootIdentity{3}) == nil {
		t.Fatal("partial/foreign handshake accepted")
	}
	if handled, _ := PropertySupervisorProcess([]string{supervisorArgument, "extra"}); !handled {
		t.Fatal("private invalid mode leaked into normal commands")
	}
	if handled, _ := PropertySupervisorProcess([]string{"helper"}); handled {
		t.Fatal("normal command entered private supervisor")
	}
}
