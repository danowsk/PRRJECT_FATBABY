package feedserver

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRoundTrip_AllFrameTypes(t *testing.T) {
	types := []FrameType{TypeHello, TypeAck, TypePing, TypeGoodbye, TypeWelcome, TypeRecord, TypePong, TypeBehind, TypeError, TypeHeartbeat}
	for _, ft := range types {
		var b bytes.Buffer
		payload := []byte(`{"x":1}`)
		if err := WriteFrame(&b, ft, payload); err != nil {
			t.Fatal(err)
		}
		gotT, gotP, err := ReadFrame(&b)
		if err != nil {
			t.Fatal(err)
		}
		if gotT != ft || string(gotP) != string(payload) {
			t.Fatalf("mismatch")
		}
	}
}
func TestReadFrame_RejectOversized(t *testing.T) {
	old := MaxFrameSize
	MaxFrameSize = 10
	defer func() { MaxFrameSize = old }()
	var b bytes.Buffer
	var h [8]byte
	binary.BigEndian.PutUint16(h[:2], Magic)
	h[2] = Version
	h[3] = byte(TypePing)
	binary.BigEndian.PutUint32(h[4:], 11)
	b.Write(h[:])
	b.Write(make([]byte, 11))
	_, _, err := ReadFrame(&b)
	if err == nil {
		t.Fatal("expected")
	}
}
func TestReadFrame_RejectBadMagic(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte{0, 0, Version, byte(TypePing), 0, 0, 0, 0})
	_, _, err := ReadFrame(&b)
	if err == nil {
		t.Fatal("expected")
	}
}
func TestWriteFrame_NoAlloc(t *testing.T) {
	payload := []byte(`{"x":1}`)
	allocs := testing.AllocsPerRun(1000, func() { var b bytes.Buffer; _ = WriteFrame(&b, TypePing, payload) })
	if allocs > 3 {
		t.Fatalf("allocs=%f", allocs)
	}
}
