package x64_encoding

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/vishen/go-brainfunk/elf"
)

// NOTE: https://www.felixcloutier.com/x86/

type Register int8

func (r Register) IsExt() bool {
	return r&8 == 8
}
func (r Register) Reg() byte {
	// TODO: I am sure there is a bit manipulation way to do this?
	if r < 8 {
		return byte(r)
	} else {
		return byte(r ^ 8)
	}
}

const (
	RAX Register = iota
	RCX
	RDX
	RBX
	RSP
	RBP
	RSI
	RDI
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
	RegNull = RAX // This is used as a replacement for op2 for 1 operand instructions.
)

type Builder struct {
	output         []byte
	currentBssSize uint32

	elfB *elf.Builder

	addrID                int
	addrIDToIndexInOutput map[int]int
}

func NewBuilder() *Builder {
	return &Builder{
		elfB:                  elf.NewBuilder(),
		addrIDToIndexInOutput: make(map[int]int),
	}
}

func (b *Builder) Build() []byte {
	return b.elfB.Build(b.output, b.currentBssSize)
}

func (b *Builder) CurrentOffset() int32 {
	return int32(len(b.output))
}

func (b *Builder) BssAdd(size uint32) uint32 {
	addr := b.currentBssSize + b.elfB.BssStartAddr()
	b.currentBssSize += size
	return addr

}

func (b *Builder) hex() string {
	return hex.EncodeToString(b.output)
}

func (b *Builder) emitREX(operand64Bit, regExt, sibIndexExt, rmExt bool) {
	var rex byte = 0x40 // REX prefix

	if operand64Bit {
		rex |= 1 << 3
	}
	if regExt {
		rex |= 1 << 2
	}
	if sibIndexExt {
		rex |= 1 << 1
	}
	if rmExt {
		rex |= 1
	}
	b.output = append(b.output, rex)
}

func (b *Builder) emitModRM(mod byte, reg byte, rm byte) {
	var modrm byte = 0x0
	modrm |= (rm | (reg << 3) | (mod << 6))
	b.output = append(b.output, modrm)
}

func (b *Builder) emitModRMWithDisplacement(op1 byte, op2 byte, displacement uint32) {
	if displacement < 128 {
		b.emitModRM(0x01, op2, op1)
		b.output = append(b.output, uint8(displacement))
	} else {
		b.emitModRM(0x02, op2, op1)
		// TODO: This seems like an inconvienient way to do this?
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, displacement)
		b.output = append(b.output, buf...)
	}
}

func (b *Builder) EmitInt(imm byte) {
	b.output = append(b.output, 0xcd, imm)
}

func (b *Builder) EmitJeqNotYetDefined() int {
	b.addrID += 1
	index := len(b.output)
	b.addrIDToIndexInOutput[b.addrID] = index
	// Fill the output with nop's and set these later.
	// https://www.reddit.com/r/compsci/comments/ljbpq/how_does_a_onepass_assembler_handle_forward_jumps/
	b.output = append(b.output, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90)
	return b.addrID
}

func (b *Builder) EmitNop() {
	b.output = append(b.output, 0x90)
}

func (b *Builder) CompleteJeq(addrID int, offset int32, length int) {
	jeqOffset := b.addrIDToIndexInOutput[addrID]
	jumpToOffset := (int(offset)) - (jeqOffset + length) // TODO: Offset This doesn't need to be int32...
	var output []byte
	if jumpToOffset >= -128 && jumpToOffset <= 127 {
		// jumpToOffset += 2 // Length of the jump instruction
		output = []byte{0x74, byte(jumpToOffset)}
	} else {
		output = []byte{0x0F, 0x10 + 0x74}
		buf := make([]byte, 4)
		// NOTE: This magic 6 is the length of this encoding. The
		// jump offset needs to be AFTER the command has executed.
		// jumpToOffset += 6 // Length of the jump instruction.
		// x := uint32(0xffffffff) - uint32(jumpToOffset-1)
		binary.LittleEndian.PutUint32(buf, uint32(jumpToOffset))
		output = append(output, buf...)
	}
	// TODO: Fix offset being int32!
	for i := 0; i < len(output); i++ {
		b.output[jeqOffset+i] = output[i]
	}
}

func (b *Builder) EmitJneBack(offset int32) int {
	/*
		// http://blog.jeff.over.bz/assembly/compilers/jit/2017/01/15/x86-assembler.html
		uint8_t *jcc_mnemonic(int32_t bytes, uint8_t *buf) {
			if (INT8_MIN <= bytes && bytes <= INT8_MAX) {
					*buf++ = byte_opcode;
					*buf++ = (int8_t)bytes;
			} else {
					*buf++ = 0x0F;
					*buf++ = byte_opcode + 0x10;
					*((int32_t *)buf) = bytes; buf += sizeof(int32_t);
			}
			return buf;
		}
	*/

	// two's complement of the distance between the current
	// instruction and the offset
	jumpToOffset := int32(len(b.output)) - offset
	outputBefore := len(b.output)
	if jumpToOffset >= -128 && jumpToOffset <= 127 {
		jumpToOffset += 2 // Length of the jump instruction
		b.output = append(b.output, 0x75, byte(0xff-(jumpToOffset-1)))
	} else {
		b.output = append(b.output, 0x0F, 0x10+0x75)
		buf := make([]byte, 4)
		// NOTE: This magic 6 is the length of this encoding. The
		// jump offset needs to be AFTER the command has executed.
		jumpToOffset += 6 // Length of the jump instruction.
		x := uint32(0xffffffff) - uint32(jumpToOffset-1)
		binary.LittleEndian.PutUint32(buf, x)
		b.output = append(b.output, buf...)
	}
	return len(b.output) - outputBefore
}

func (b *Builder) EmitJmpForwardRelative(offset int32) {
	b.output = append(b.output, 0xeb, byte(b.CurrentOffset()+offset))
}

func (b *Builder) EmitCall(offset int32) {
	// two's complement of the distance between the current
	// instruction and the offset
	toOffset := int32(len(b.output)) - offset
	toOffset += 5 // Length of the jump instruction
	x := uint32(0xffffffff) - uint32(toOffset-1)
	b.output = append(b.output, 0xE8)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, x)
	b.output = append(b.output, buf...)
}

func (b *Builder) EmitIncReg(src Register) {
	b.emitREX(true, false, false, src.IsExt())
	b.output = append(b.output, 0xFF)
	b.emitModRM(0x03, 0, src.Reg())
}

func (b *Builder) EmitIncMem(src Register, displacement uint32) {
	b.emitREX(true, false, false, src.IsExt())
	b.output = append(b.output, 0xFF)
	b.emitModRMWithDisplacement(src.Reg(), 0, displacement)
}

func (b *Builder) EmitDecReg(src Register) {
	b.emitREX(true, false, false, src.IsExt())
	b.output = append(b.output, 0xFF)
	b.emitModRM(0x03, 0x01, src.Reg())
}

func (b *Builder) EmitDecMem(src Register, displacement uint32) {
	b.emitREX(true, false, false, src.IsExt())
	b.output = append(b.output, 0xFF)
	b.emitModRMWithDisplacement(src.Reg(), 1, displacement)
}

func (b *Builder) EmitMovRegImm(src Register, imm uint32) {
	b.emitREX(true, false, false, src.IsExt())
	b.output = append(b.output, 0xC7)
	b.emitModRM(0x03, 0, src.Reg())
	// TODO: Move to function
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, imm)
	b.output = append(b.output, buf...)
}

func (b *Builder) EmitMovRegReg(src, dest Register) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	b.output = append(b.output, 0x89)
	b.emitModRM(0x3, dest.Reg(), src.Reg())
}

func (b *Builder) EmitMovMemReg(src, dest Register, displacement uint32) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	b.output = append(b.output, 0x89)
	b.emitModRMWithDisplacement(src.Reg(), dest.Reg(), displacement)
}

func (b *Builder) EmitMovRegMem(src, dest Register, displacement uint32) {
	b.emitREX(true, src.IsExt(), false, dest.IsExt())
	b.output = append(b.output, 0x8b)
	if displacement == 0 {
		b.emitModRM(0x00, src.Reg(), dest.Reg())
	} else {
		b.emitModRMWithDisplacement(dest.Reg(), src.Reg(), displacement)
	}
}

// Add instructions
func (b *Builder) EmitAddRegImm(src Register, imm uint32) {

	b.emitREX(true, false, false, src.IsExt())
	// NOTE: src == RAX and imm == 32-bit, then special case
	if src == RAX && imm >= 128 {
		// REX.W + 05 id	ADD RAX, imm32
		b.output = append(b.output, 0x05)
	} else {
		if imm < 128 {
			// REX.W + 83 /0 ib    ADD r/m64, imm8
			b.output = append(b.output, 0x83)
		} else {
			// REX.W + 81 /0 id	ADD r/m64, imm32
			b.output = append(b.output, 0x81)
		}
		b.emitModRM(0x03, 0, src.Reg())
	}
	// TODO: This is very similar to the displacement checking if
	// imm8 or imm32, maybe refactor if possible?
	if imm >= 128 {
		// TODO: Move to function
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, imm)
		b.output = append(b.output, buf...)
	} else {
		b.output = append(b.output, uint8(imm))
	}
}

func (b *Builder) EmitAddRegReg(src, dest Register) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 01 /r	ADD r/m64, r64
	b.output = append(b.output, 0x01)
	b.emitModRM(0x3, dest.Reg(), src.Reg())
}

func (b *Builder) EmitAddMemReg(src, dest Register, displacement uint32) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 01 /r	ADD r/m64, r64
	b.output = append(b.output, 0x01)
	if displacement == 0 {
		b.emitModRM(0x00, dest.Reg(), src.Reg())
	} else {
		b.emitModRMWithDisplacement(src.Reg(), dest.Reg(), displacement)
	}
}

func (b *Builder) EmitAddRegMem(src, dest Register, displacement uint32) {
	// TODO: Why on earth are these around the other way than every other
	// instruction variation??????? Seems to be the same for everything in
	// this variation?
	b.emitREX(true, src.IsExt(), false, dest.IsExt())
	// REX.W + 03 /r	ADD r64, r/m64
	b.output = append(b.output, 0x03)
	if displacement == 0 {
		b.emitModRM(0x00, src.Reg(), dest.Reg())
	} else {
		b.emitModRMWithDisplacement(dest.Reg(), src.Reg(), displacement)
	}
}

// Sub instruction
func (b *Builder) EmitSubRegImm(src Register, imm uint32) {

	b.emitREX(true, false, false, src.IsExt())
	// NOTE: src == RAX and imm == 32-bit, then special case
	if src == RAX && imm >= 128 {
		// REX.W + 2D id	SUB RAX, imm32
		b.output = append(b.output, 0x2d)
	} else {
		if imm < 128 {
			// REX.W + 83 /5 ib	SUB r/m64, imm8
			b.output = append(b.output, 0x83)
			b.emitModRM(0x03, 0x05, src.Reg())
		} else {
			// REX.W + 81 /5 id	SUB r/m64, imm32
			b.output = append(b.output, 0x81)
			b.emitModRM(0x03, 0x05, src.Reg())
		}
	}
	// TODO: This is very similar to the displacement checking if
	// imm8 or imm32, maybe refactor if possible?
	if imm >= 128 {
		// TODO: Move to function
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, imm)
		b.output = append(b.output, buf...)
	} else {
		b.output = append(b.output, uint8(imm))
	}
}

func (b *Builder) EmitSubRegReg(src, dest Register) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 29 /r	SUB r/m64, r64
	b.output = append(b.output, 0x29)
	b.emitModRM(0x3, dest.Reg(), src.Reg())
}

func (b *Builder) EmitSubMemReg(src, dest Register, displacement uint32) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 29 /r	SUB r/m64, r64
	b.output = append(b.output, 0x29)
	if displacement == 0 {
		b.emitModRM(0x00, dest.Reg(), src.Reg())
	} else {
		b.emitModRMWithDisplacement(src.Reg(), dest.Reg(), displacement)
	}
}

func (b *Builder) EmitSubRegMem(src, dest Register, displacement uint32) {
	// TODO: Why on earth are these around the other way than every other
	// instruction variation??????? Seems to be the same for everything in
	// this variation?
	b.emitREX(true, src.IsExt(), false, dest.IsExt())
	// REX.W + 2B /r	SUB r64, r/m64
	b.output = append(b.output, 0x2b)
	if displacement == 0 {
		b.emitModRM(0x00, src.Reg(), dest.Reg())
	} else {
		b.emitModRMWithDisplacement(dest.Reg(), src.Reg(), displacement)
	}
}

// Cmp instruction
func (b *Builder) EmitCmpMemImm(src Register, imm uint32) {
	b.emitREX(true, false, false, src.IsExt())
	if imm < 128 {
		// REX.W + 83 /7 ib	   CMP r/m64, imm8
		b.output = append(b.output, 0x83)
		b.emitModRM(0x00, 0x07, src.Reg())
	} else {
		// REX.W + 81 /7 id	CMP r/m64, imm32
		b.output = append(b.output, 0x81)
		b.emitModRM(0x00, 0x07, src.Reg())
	}
	// TODO: This is very similar to the displacement checking if
	// imm8 or imm32, maybe refactor if possible?
	if imm >= 128 {
		// TODO: Move to function
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, imm)
		b.output = append(b.output, buf...)
	} else {
		b.output = append(b.output, uint8(imm))
	}
}

func (b *Builder) EmitCmpRegImm(src Register, imm uint32) {
	b.emitREX(true, false, false, src.IsExt())
	// NOTE: src == RAX and imm == 32-bit, then special case
	if src == RAX && imm >= 128 {
		// REX.W + 3D id	CMP RAX, imm32
		b.output = append(b.output, 0x3d)
	} else {
		if imm < 128 {
			// REX.W + 83 /7 ib	   CMP r/m64, imm8
			b.output = append(b.output, 0x83)
			b.emitModRM(0x03, 0x07, src.Reg())
		} else {
			// REX.W + 81 /7 id	CMP r/m64, imm32
			b.output = append(b.output, 0x81)
			b.emitModRM(0x03, 0x07, src.Reg())
		}
	}
	// TODO: This is very similar to the displacement checking if
	// imm8 or imm32, maybe refactor if possible?
	if imm >= 128 {
		// TODO: Move to function
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, imm)
		b.output = append(b.output, buf...)
	} else {
		b.output = append(b.output, uint8(imm))
	}
}

func (b *Builder) EmitCmpRegReg(src, dest Register) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 39 /r	CMP r/m64,r64
	b.output = append(b.output, 0x39)
	b.emitModRM(0x3, dest.Reg(), src.Reg())
}

func (b *Builder) EmitCmpMemReg(src, dest Register, displacement uint32) {
	b.emitREX(true, dest.IsExt(), false, src.IsExt())
	// REX.W + 39 /r	CMP r/m64,r64
	b.output = append(b.output, 0x39)
	if displacement == 0 {
		b.emitModRM(0x00, dest.Reg(), src.Reg())
	} else {
		b.emitModRMWithDisplacement(src.Reg(), dest.Reg(), displacement)
	}
}

func (b *Builder) EmitCmpRegMem(src, dest Register, displacement uint32) {
	// TODO: Why on earth are these around the other way than every other
	// instruction variation??????? Seems to be the same for everything in
	// this variation?
	b.emitREX(true, src.IsExt(), false, dest.IsExt())
	// REX.W + 3B /r	CMP r64, r/m64
	b.output = append(b.output, 0x3b)
	if displacement == 0 {
		b.emitModRM(0x00, src.Reg(), dest.Reg())
	} else {
		b.emitModRMWithDisplacement(dest.Reg(), src.Reg(), displacement)
	}
}

func (b *Builder) EmitRet() {
	b.output = append(b.output, 0xc3)
}
