package bitpack

// readBit returns the bit at stream position pos.
//
// LittleEndian (bigEndian=false):
//
//	bit 0 of stream = LSB (bit 0) of byte 0
//	bit 8 of stream = LSB (bit 0) of byte 1
//
// BigEndian (bigEndian=true):
//
//	bit 0 of stream = MSB (bit 7) of byte 0
//	bit 8 of stream = MSB (bit 7) of byte 1
func readBit(data []byte, pos int, bigEndian bool) byte {
	byteIdx := pos / 8
	bitIdx := pos % 8
	if bigEndian {
		bitIdx = 7 - bitIdx
	}
	return (data[byteIdx] >> bitIdx) & 1
}

// readBits reads n consecutive bits from the stream starting at pos and
// assembles them into a uint64.
//
// LittleEndian: first bit read becomes LSB of the returned value.
// BigEndian:    first bit read becomes MSB of the returned value.
func readBits(data []byte, pos, n int, bigEndian bool) uint64 {
	var result uint64
	for i := 0; i < n; i++ {
		bit := uint64(readBit(data, pos+i, bigEndian))
		if bigEndian {
			result = (result << 1) | bit
		} else {
			result |= bit << i
		}
	}
	return result
}

// writeBit sets or clears the bit at stream position pos.
func writeBit(data []byte, pos int, bit byte, bigEndian bool) {
	byteIdx := pos / 8
	bitIdx := pos % 8
	if bigEndian {
		bitIdx = 7 - bitIdx
	}
	if bit == 1 {
		data[byteIdx] |= 1 << bitIdx
	} else {
		data[byteIdx] &^= 1 << bitIdx
	}
}

// writeBits writes n bits of value into the stream starting at pos.
//
// LittleEndian: LSB of value is written to the lowest stream position.
// BigEndian:    MSB of value is written to the lowest stream position.
func writeBits(data []byte, pos, n int, value uint64, bigEndian bool) {
	for i := 0; i < n; i++ {
		var bit byte
		if bigEndian {
			bit = byte((value >> (n - 1 - i)) & 1)
		} else {
			bit = byte((value >> i) & 1)
		}
		writeBit(data, pos+i, bit, bigEndian)
	}
}
