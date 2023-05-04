package jpegsl

import (
	"bytes"
	"log"
)

type Bitstream struct {
	dataStream   *bytes.Reader
	bytePosition int
	byteBuffer   byte
}

func NewBitstream(data []byte) *Bitstream {
	var bs = new(Bitstream)
	bs.dataStream = bytes.NewReader(data)
	bs.bytePosition = 0
	bs.byteBuffer = 0
	return bs
}

func (bs *Bitstream) bit() int {
	if bs.bytePosition == 0 {
		bs.byteBuffer = read1Byte(bs.dataStream)
		if bs.byteBuffer == 0xff {
			bs.dataStream.ReadByte()
		}

		bs.bytePosition = 8
	}
	bs.bytePosition -= 1
	return int((bs.byteBuffer >> bs.bytePosition) & 1)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func read1Byte(reader *bytes.Reader) byte {
	b, err := reader.ReadByte()
	if err != nil {
		log.Panic(err)
	}
	return b
}

func read2Bytes(reader *bytes.Reader) uint16 {
	b0, err := reader.ReadByte()
	if err != nil {
		log.Panic(err)
	}

	b1, err := reader.ReadByte()
	if err != nil {
		log.Panic(err)
	}

	return uint16(b0)<<8 | uint16(b1)
}

func (bs *Bitstream) bits(length int) int {
	nextLength := min(bs.bytePosition, length)
	length -= nextLength
	bs.bytePosition -= nextLength
	currentBits := int(bs.byteBuffer>>bs.bytePosition) & int((1<<nextLength)-1)

	for length > 0 {
		bs.byteBuffer = read1Byte(bs.dataStream)

		if bs.byteBuffer == 0xff {
			bs.dataStream.ReadByte()
		}

		nextLength = min(8, length)
		length -= nextLength
		bs.bytePosition = 8 - nextLength

		currentBits <<= nextLength
		currentBits |= int(bs.byteBuffer>>bs.bytePosition) & int((1<<nextLength)-1)
	}

	return int(currentBits)
}
