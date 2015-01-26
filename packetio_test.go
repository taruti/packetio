package packetio

import (
	"bytes"
	"io"
	"testing"
)

type A struct {
	s string
}

func (a *A) Size() int {
	return len(a.s)
}
func (a *A) MarshalTo(bs []byte) (int, error) {
	n := copy(bs, a.s)
	return n, nil
}
func (a *A) Unmarshal(bs []byte) error {
	a.s = string(bs)
	return nil
}

var theA = &A{"my test is this a string or an A"}

type Fixed struct {
	f [32]byte
}

func (a *Fixed) Size() int {
	return len(a.f)
}
func (a *Fixed) MarshalTo(bs []byte) (int, error) {
	n := copy(bs, a.f[:])
	return n, nil
}
func (a *Fixed) Unmarshal(bs []byte) error {
	copy(a.f[:], bs)
	return nil
}

var theFixed = &Fixed{}

func TestWritePacket(t *testing.T) {
	testWritePacket(1, t)
}

func BenchmarkWritePacket(b *testing.B) {
	testWritePacket(b.N, b)
}

func testWritePacket(n int, t testing.TB) {
	b := new(bytes.Buffer)
	var wp PacketWriter
	wp.Init(b)
	for i := 0; i < n; i++ {
		b.Reset()
		_, e := wp.WritePacket(1, theA)
		if e != nil {
			t.Fatal("WritePacket", e)
		}
	}
}

func single(ty byte, m Marshaller) string {
	b := new(bytes.Buffer)
	var wp PacketWriter
	wp.Init(b)
	wp.WritePacket(ty, m)
	return b.String()
}

type simpleReader struct {
	s string
	i int
}

func (sr *simpleReader) Read(bs []byte) (int, error) {
	if len(sr.s) <= sr.i {
		return 0, io.EOF
	}
	n := copy(bs, sr.s[sr.i:])
	sr.i += n
	return n, nil
}
func (sr *simpleReader) Reset() {
	sr.i = 0
}

func TestReadPacket(t *testing.T) {
	testReadPacket(1, t)
}

func BenchmarkReadPacket(b *testing.B) {
	testReadPacket(b.N, b)
}

func testReadPacket(n int, t testing.TB) {
	sr := &simpleReader{single(1, theA), 0}
	var wr PacketReader
	wr.Init(sr, []Unmarshaller{nil, &A{}, &Fixed{}})
	for i := 0; i < n; i++ {
		_, e := wr.ReadPacket()
		if e != nil {
			t.Fatal("ReadPacket", e)
		}
		sr.Reset()
	}
}

func TestReadPacketFixed(t *testing.T) {
	testReadPacketFixed(1, t)
}

func BenchmarkReadPacketFixed(b *testing.B) {
	testReadPacketFixed(b.N, b)
}

func testReadPacketFixed(n int, t testing.TB) {
	sr := &simpleReader{single(2, theFixed), 0}
	var wr PacketReader
	wr.Init(sr, []Unmarshaller{nil, &A{}, &Fixed{}})
	for i := 0; i < n; i++ {
		_, e := wr.ReadPacket()
		if e != nil {
			t.Fatal("ReadPacket", e)
		}
		sr.Reset()
	}
}
