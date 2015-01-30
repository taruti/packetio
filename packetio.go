/*Package packetio provides length delimeted typed serialization compatible with protobuf.

packetio provides an easy way to delimit messages with length and a type. This makes
serializing e.g. protobuf (gogoprotobuf) messages much easier. No dependency on
protobuf however.

*/
package packetio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

// Marshaller is an interface for encoding objects that is compatible with gogoprotobuf.
type Marshaller interface {
	Size() int
	MarshalTo([]byte) (int, error)
}

// Unmarshaller is an interface for decoding objects that is compatible with gogoprotobuf.
type Unmarshaller interface {
	Unmarshal(data []byte) error
}

// ErrTooLarge is returned when the packet is too large to be written by WritePacket.
var ErrTooLarge = errors.New("too large packet")

// ErrNoUnmarshaller is returned when the packet was of a type with no Unmarshaller
// given. The unknown packet has been read from the stream and ReadPacket can
// be called for the following packets.
var ErrNoUnmarshaller = errors.New("no unmarshaller for type")

const packetWriterExtra = 8

// PacketWriter writes packets that are Marshallers into a io.Writer.
type PacketWriter struct {
	w   io.Writer
	tmp []byte
}

// Init initializes a PacketWriter with the destination io.Writer.
func (pw *PacketWriter) Init(w io.Writer) {
	pw.w = w
}

// WritePacket writes the Marshaller to the destination with a length prefix and
// a single byte type field.
func (pw *PacketWriter) WritePacket(packetType byte, msg Marshaller) (int, error) {
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
		return 0, ErrTooLarge
	}

	binary.BigEndian.PutUint32(pw.tmp[4:], uint32(n))
	pw.tmp[4] = packetType

	buf := pw.tmp[4 : packetWriterExtra+n]
	wn, e := pw.w.Write(buf)
	return wn, e
}

// PacketReader is for decoding values encoded with PacketWriter from an io.Reader.
type PacketReader struct {
	br    *bufio.Reader
	umarr []Unmarshaller
	tmp   []byte
	b0    [4]byte
}

// Init initializes the PacketReader with the source io.Reader and a slice
// of Unmarshaller values, with each index used to decode values of that
// packetType. Nil-values are permitted, and if the type is out of range
// or the corresponding Unmarshaller is nil an error is returned.
func (pr *PacketReader) Init(rd io.Reader, uvs []Unmarshaller) {
	pr.br = bufio.NewReader(rd)
	pr.umarr = uvs
}

// ReadPacket reads a packet from the stream using the unmarshallers
// passed to Init. Note that the Unmarshaller itself is used for decoding
// and returned rather than making a copy.
func (pr *PacketReader) ReadPacket() (Unmarshaller, error) {
	buf := pr.b0[:]
	_, e := io.ReadFull(pr.br, buf)
	if e != nil {
		return nil, e
	}
	st := int(buf[0])
	buf[0] = 0
	next := int(binary.BigEndian.Uint32(buf))
	if next > len(pr.tmp) {
		freeiobuffer(pr.tmp)
		pr.tmp = newiobuffer(next)
	}
	_, e = io.ReadFull(pr.br, pr.tmp[:next])
	if e != nil {
		return nil, e
	}
	if st > len(pr.umarr) || pr.umarr[st] == nil {
		return nil, ErrNoUnmarshaller
	}
	res := pr.umarr[st]
	e = res.Unmarshal(pr.tmp[:next])
	return res, e
}

func freeiobuffer([]byte) {}
func newiobuffer(size int) []byte {
	if size < 8*1024 {
		size = 8 * 1024
	}
	return make([]byte, size)
}
