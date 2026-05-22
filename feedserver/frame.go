package feedserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const Magic = uint16(0xFBAB)
const Version = uint8(0x01)
const DefaultMaxFrameSize = 1 << 20

const FromSeqFullBackfill = uint64(0)
const FromSeqRealtimeOnly = uint64(math.MaxUint64)

var MaxFrameSize uint32 = DefaultMaxFrameSize

type FrameType uint8

const (
	TypeHello     FrameType = 0x01
	TypeAck       FrameType = 0x02
	TypePing      FrameType = 0x03
	TypeGoodbye   FrameType = 0x04
	TypeWelcome   FrameType = 0x11
	TypeRecord    FrameType = 0x12
	TypePong      FrameType = 0x13
	TypeBehind    FrameType = 0x14
	TypeError     FrameType = 0x15
	TypeHeartbeat FrameType = 0x16
)

func WriteFrame(w io.Writer, ft FrameType, payload []byte) error {
	var header [8]byte
	binary.BigEndian.PutUint16(header[0:2], Magic)
	header[2] = Version
	header[3] = byte(ft)
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}

func ReadFrame(r io.Reader) (FrameType, []byte, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}
	if binary.BigEndian.Uint16(header[0:2]) != Magic {
		return 0, nil, errors.New("bad magic")
	}
	if header[2] != Version {
		return 0, nil, errors.New("bad version")
	}
	l := binary.BigEndian.Uint32(header[4:8])
	if l > MaxFrameSize {
		return 0, nil, fmt.Errorf("frame too large: %d", l)
	}
	payload := make([]byte, l)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return FrameType(header[3]), payload, nil
}
