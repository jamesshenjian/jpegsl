package jpegsl

import (
	"bytes"
	"fmt"
	"io"
)

type Decoder struct {
	bitstream            *Bitstream
	dataStream           *bytes.Reader
	huffmanTrees         map[byte]*HuffmanNode
	huffmanTreesSelected map[int]*HuffmanNode

	precision       byte
	predictor       byte
	lines           uint16
	samples         uint16
	components      int
	componentIndex  map[byte]int
	samplingFactorH []byte
	samplingFactorV []byte
}

var MARKER_SOF3 uint16 = 0xffc3
var MARKER_DHT uint16 = 0xffc4
var MARKER_SOI uint16 = 0xffd8
var MARKER_SOS uint16 = 0xffda

func NewDecoder(data []byte) *Decoder {
	decoder := new(Decoder)
	decoder.bitstream = NewBitstream(data)
	decoder.dataStream = decoder.bitstream.dataStream
	return decoder
}

func (decoder *Decoder) buildTree() int {
	tableLength := 0
	tableId := read1Byte(decoder.dataStream)

	if decoder.huffmanTrees == nil {
		decoder.huffmanTrees = make(map[byte]*HuffmanNode)
	}

	decoder.huffmanTrees[tableId] = NewHuffmanNode(decoder.bitstream)

	codeLengthArray := make([]byte, 16)
	vals := make([]byte, 16)

	for i := 0; i < 16; i++ {
		codeLengthArray[i] = read1Byte(decoder.dataStream)
		tableLength += int(codeLengthArray[i])
	}

	k := 0
	for i := 0; i < 16; i++ {
		for j := 0; j < int(codeLengthArray[i]); j++ {
			vals[k] = read1Byte(decoder.dataStream)
			decoder.huffmanTrees[tableId].mostLeft(i + 1).value = int(vals[k])
			k++
		}
	}

	h := decoder.huffmanTrees[tableId]

	var x, code uint32
	for i := uint32(0); i < lutSize; i++ {
		code <<= 1
		for j := 0; j < int(codeLengthArray[i]); j++ {
			// The codeLength is 1+i, so shift code by 8-(1+i) to
			// calculate the high bits for every 8-bit sequence
			// whose codeLength's high bits matches code.
			// The high 8 bits of lutValue are the encoded value.
			// The low 8 bits are 1 plus the codeLength.
			base := uint8(code << (7 - i))
			lutValue := uint16(vals[x])<<8 | uint16(2+i)
			for k := uint8(0); k < 1<<(7-i); k++ {
				h.lut[base|k] = lutValue
			}
			code++
			x++
		}
	}

	//fmt.Print(h.lut)
	return tableLength + 17
}

func (decoder *Decoder) decodeDHT(length int64) {
	for length > 0 {
		length -= int64(decoder.buildTree())
	}
}

func (decoder *Decoder) decodeSOF3(length int64) {
	decoder.precision = read1Byte(decoder.dataStream)
	decoder.lines = read2Bytes(decoder.dataStream)
	decoder.samples = read2Bytes(decoder.dataStream)
	decoder.components = int(read1Byte(decoder.dataStream))
	decoder.componentIndex = make(map[byte]int, decoder.components)
	decoder.samplingFactorH = make([]byte, decoder.components)
	decoder.samplingFactorV = make([]byte, decoder.components)

	for i := 0; i < decoder.components; i++ {
		component := read1Byte(decoder.dataStream)
		samplingFactor := read1Byte(decoder.dataStream)
		read1Byte(decoder.dataStream)
		decoder.componentIndex[component] = i
		decoder.samplingFactorH[i] = samplingFactor >> 4
		decoder.samplingFactorV[i] = samplingFactor & 0xf
	}

	decoder.dataStream.Seek(length-int64(6+decoder.components*3), io.SeekCurrent)
}

func (decoder *Decoder) decodeSOS(length int64) {
	components := read1Byte(decoder.dataStream)
	if decoder.huffmanTreesSelected == nil {
		decoder.huffmanTreesSelected = make(map[int]*HuffmanNode)
	}

	for i := 0; i < int(components); i++ {
		component := read1Byte(decoder.dataStream)
		treeSelection := read1Byte(decoder.dataStream)
		decoder.huffmanTreesSelected[decoder.componentIndex[component]] = decoder.huffmanTrees[treeSelection>>4]
	}

	decoder.predictor = read1Byte(decoder.dataStream)

	decoder.dataStream.Seek(length-int64(2+components*2), io.SeekCurrent)
}

func (decoder *Decoder) decodeHeader() {
	marker := read2Bytes(decoder.dataStream)

	if marker != MARKER_SOI {
		return
	}

	done := false
	for !done {
		marker := read2Bytes(decoder.dataStream)
		length := read2Bytes(decoder.dataStream) - 2

		switch marker {
		case MARKER_SOF3:
			decoder.decodeSOF3(int64(length))
		case MARKER_DHT:
			decoder.decodeDHT(int64(length))
		case MARKER_SOS:
			decoder.decodeSOS(int64(length))
			done = true
		default:
			decoder.dataStream.Seek(int64(length), io.SeekCurrent)
		}
	}
}

func (decoder *Decoder) decodeDiff(node *HuffmanNode) int {

	length := node.decode(true)
	if length < 0 {
		fmt.Print(length)
	}

	if length == 0 {
		return 0
	}

	if length == 16 {
		return -32768
	}

	diff := int(decoder.bitstream.bits(length))

	if (diff & (1 << (length - 1))) == 0 {
		diff -= (1 << length) - 1
	}

	return diff
}

// Decode bytes to int array. pass signed as true if the pixel value are signed and false otherwise
// For dicom pixel data parsing, get the value of tag PixelRepresentation. 1 means signed and 0 means unsigned
// Also check the tag PixelPaddingValue, which can be used to pad the image. Those pixels should be treated correctly
func Decode(data []byte, signed bool) []int {
	decoder := NewDecoder(data)
	decoder.decodeHeader()

	width := int(decoder.samples) * decoder.components
	imageArray := make([]int, int(decoder.lines)*width)
	stripeSize := width

	for i := 0; i < decoder.components; i++ {
		imageArray[i] = decoder.decodeDiff(decoder.huffmanTreesSelected[i])
		if !signed {
			imageArray[i] += (1 << (decoder.precision - 1))
		} else {
			imageArray[i] -= (1 << (decoder.precision - 1))
		}
	}

	for x := decoder.components; x < int(decoder.samples)*decoder.components; x += decoder.components {
		for i := 0; i < decoder.components; i++ {
			imageArray[x+i] = decoder.decodeDiff(decoder.huffmanTreesSelected[i]) + imageArray[x+i-decoder.components]
		}
	}

	offset := stripeSize

	for y := 1; y < int(decoder.lines); y++ {
		for i := 0; i < decoder.components; i++ {
			imageArray[offset+i] = decoder.decodeDiff(decoder.huffmanTreesSelected[i]) + imageArray[offset+i-stripeSize]
		}
		for x := decoder.components; x < int(decoder.samples)*decoder.components; x += decoder.components {
			for i := 0; i < decoder.components; i++ {
				predictor := 0
				switch decoder.predictor {
				case 1:
					predictor = imageArray[offset+x+i-decoder.components]
				case 6:
					predictor = imageArray[offset+x+i-stripeSize] + ((imageArray[offset+x+i-decoder.components] -
						imageArray[offset+x+i-decoder.components-stripeSize]) >> 1)
				}
				imageArray[offset+x+i] = predictor + decoder.decodeDiff(decoder.huffmanTreesSelected[i])
			}
		}
		offset += stripeSize
	}

	return imageArray
}
