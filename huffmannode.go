package jpegsl

const (
	lutSize = 8
)

type HuffmanNode struct {
	bitstream *Bitstream
	nodes     [2]*HuffmanNode
	value     int
	lut       [1 << lutSize]uint16
}

const (
	undefined = -1
)

func NewHuffmanNode(bitstream *Bitstream) *HuffmanNode {
	hn := new(HuffmanNode)
	hn.bitstream = bitstream
	hn.nodes[0], hn.nodes[1] = nil, nil
	hn.value = undefined
	for i := range hn.lut {
		hn.lut[i] = 0
	}
	return hn
}

func (hn *HuffmanNode) mostLeft(level int) *HuffmanNode {
	if hn.value != undefined {
		return nil
	}

	if level == 0 {
		return hn
	}

	for i := 0; i < 2; i++ {
		var nextNode *HuffmanNode = nil

		if hn.nodes[i] == nil {
			hn.nodes[i] = NewHuffmanNode(hn.bitstream)
		}

		if nextNode = hn.nodes[i].mostLeft(level - 1); nextNode != nil {
			return nextNode
		}
	}

	return nil
}

func (hn *HuffmanNode) decode(isRoot bool) int {
	if isRoot {
		//try fast forward using the lookup table for code of length <=8
		//test on a 512x512 pixel image shows improved performance by ~30%
		nextByte := hn.bitstream.tryByte()
		if hn.lut[nextByte] > 0 {
			codeLen := hn.lut[nextByte]&0xff - 1
			hn.bitstream.advance(int(codeLen))
			return int((hn.lut[nextByte] >> 8) & 0xff)
		}
	}

	nextNode := hn.nodes[hn.bitstream.bit()]

	if nextNode == nil {
		return 16
	}

	if nextNode.value != undefined {
		return nextNode.value
	}

	return nextNode.decode(false)
}
