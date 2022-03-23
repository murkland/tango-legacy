package asm

func NOP() []byte {
	return []byte{0xc0, 0x46}
}

func SVC(imm byte) []byte {
	return []byte{imm, 0xdf}
}

func Flatten(instrs ...[]byte) []byte {
	var buf []byte
	for _, v := range instrs {
		buf = append(buf, v...)
	}
	return buf
}
