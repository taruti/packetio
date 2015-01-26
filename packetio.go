package packetio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

type Marshaler interface {
	Size() int
	MarshalTo([]byte) (int, error)
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}

const packetWriterExtra = 8

type PacketWriter struct {
	w   io.Writer
	tmp []byte
}

func (pw *PacketWriter) Init(w io.Writer) {
	pw.w = w
}

func (pw *PacketWriter) WritePacket(packetType byte, msg Marshaler) (int, error) {
	siz := msg.Size()
	if siz+packetWriterExtra > len(pw.tmp) {
		freeiobuffer(pw.tmp)
		pw.tmp = newiobuffer(siz + packetWriterExtra)
	}
	n, e := msg.MarshalTo(pw.tmp[packetWriterExtra:])
	if e != nil {
		return 0, e
	}

	if n > 0xFFffFF {
		return 0, errors.New("WritePacket: too large packet")
	}

	binary.BigEndian.PutUint32(pw.tmp[4:], uint32(n))
	pw.tmp[4] = packetType

	buf := pw.tmp[4 : packetWriterExtra+n]
	wn, e := pw.w.Write(buf)
	return wn, e
}

type PacketReader struct {
	br    *bufio.Reader
	umarr []Unmarshaler
	tmp   []byte
	b0    [4]byte
}

func (pr *PacketReader) Init(rd io.Reader, uvs []Unmarshaler) {
	pr.br = bufio.NewReader(rd)
	pr.umarr = uvs
}

func (pr *PacketReader) ReadPacket() (Unmarshaler, error) {
	buf := pr.b0[:]
	_, e := io.ReadFull(pr.br, buf)
	if e != nil {
		return nil, e
	}
	st := int(buf[0])
	buf[0] = 0
	next := int(binary.BigEndian.Uint32(buf))
	if st > len(pr.umarr) || pr.umarr[st] == nil {
		return nil, errors.New("No unmarshaller for type")
	}
	if next > len(pr.tmp) {
		freeiobuffer(pr.tmp)
		pr.tmp = newiobuffer(next)
	}
	_, e = io.ReadFull(pr.br, pr.tmp[:next])
	if e != nil {
		return nil, e
	}
	res := pr.umarr[st]
	res.Unmarshal(pr.tmp)
	return res, nil
}

func freeiobuffer([]byte) {}
func newiobuffer(size int) []byte {
	if size < 8*1024 {
		size = 8 * 1024
	}
	return make([]byte, size)
}
