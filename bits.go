package nibble

// readBits reads n bits from stream position pos.
func readBits(data []byte, pos, n int, bigEndian bool) uint64 {
	if bigEndian {
		return readBitsBE(data, pos, n)
	}
	return readBitsLE(data, pos, n)
}

// writeBits writes n bits of value into data at stream position pos.
func writeBits(data []byte, pos, n int, value uint64, bigEndian bool) {
	if bigEndian {
		writeBitsBE(data, pos, n, value)
	} else {
		writeBitsLE(data, pos, n, value)
	}
}

// readBitsLE reads n bits starting at stream bit-position pos.
// Processes up to 8 bits per iteration instead of 1, eliminating
// per-bit division and modulo in the hot path.
// First bit consumed → LSB of returned value.
func readBitsLE(data []byte, pos, n int) uint64 {
	var result uint64
	done := 0
	for done < n {
		byteIdx := (pos + done) / 8
		bitInByte := (pos + done) % 8
		chunk := 8 - bitInByte // bits remaining in this byte
		if chunk > n-done {
			chunk = n - done
		}
		mask := uint64((1 << chunk) - 1)
		result |= ((uint64(data[byteIdx]) >> bitInByte) & mask) << done
		done += chunk
	}
	return result
}

// readBitsBE reads n bits starting at stream bit-position pos in BigEndian
// mode (stream bit 0 → MSB of byte 0).
// First bit consumed → MSB of returned value.
func readBitsBE(data []byte, pos, n int) uint64 {
	var result uint64
	done := 0
	for done < n {
		byteIdx := (pos + done) / 8
		bitInByte := (pos + done) % 8
		physBit := 7 - bitInByte   // physical bit within the byte (0=LSB)
		chunk := physBit + 1       // bits available from here toward LSB
		if chunk > n-done {
			chunk = n - done
		}
		shift := physBit - chunk + 1
		mask := uint64((1 << chunk) - 1)
		result = (result << chunk) | ((uint64(data[byteIdx]) >> shift) & mask)
		done += chunk
	}
	return result
}

// writeBitsLE writes n bits of value into data starting at stream bit-position
// pos. LSB of value → lowest stream position.
func writeBitsLE(data []byte, pos, n int, value uint64) {
	done := 0
	for done < n {
		byteIdx := (pos + done) / 8
		bitInByte := (pos + done) % 8
		chunk := 8 - bitInByte
		if chunk > n-done {
			chunk = n - done
		}
		mask := byte((1 << chunk) - 1)
		bits := byte((value >> done) & uint64(mask))
		data[byteIdx] = (data[byteIdx] &^ (mask << bitInByte)) | (bits << bitInByte)
		done += chunk
	}
}

// writeBitsBE writes n bits of value into data starting at stream bit-position
// pos in BigEndian mode. MSB of value → lowest stream position.
func writeBitsBE(data []byte, pos, n int, value uint64) {
	done := 0
	for done < n {
		byteIdx := (pos + done) / 8
		bitInByte := (pos + done) % 8
		physBit := 7 - bitInByte
		chunk := physBit + 1
		if chunk > n-done {
			chunk = n - done
		}
		shift := physBit - chunk + 1
		mask := byte((1 << chunk) - 1)
		bits := byte((value >> (n - done - chunk)) & uint64(mask))
		data[byteIdx] = (data[byteIdx] &^ (mask << shift)) | (bits << shift)
		done += chunk
	}
}
