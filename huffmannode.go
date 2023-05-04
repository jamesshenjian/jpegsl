package jpegsl

type HuffmanNode struct {
	bitstream *Bitstream
	nodes     [2]*HuffmanNode
	value     int
}

const (
	undefined = -1
)

func NewHuffmanNode(bitstream *Bitstream) *HuffmanNode {
	hn := new(HuffmanNode)
	hn.bitstream = bitstream
	hn.nodes[0], hn.nodes[1] = nil, nil
	hn.value = undefined
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

func (hn *HuffmanNode) decode() int {
	nextNode := hn.nodes[hn.bitstream.bit()]

	if nextNode == nil {
		return 16
	}

	if nextNode.value != undefined {
		return nextNode.value
	}

	return nextNode.decode()
}
