package elf

import (
	"encoding/binary"
)

const (
	virtualStartAddress    uint32 = 0x400000
	bssVirtualStartAddress uint32 = 0x600000
	alignment              uint32 = 0x200000
)

type Builder struct {
	o []byte
}

func NewBuilder() *Builder {
	return &Builder{
		o: make([]byte, 0, 1024),
	}
}

func (b *Builder) BssStartAddr() uint32 {
	return bssVirtualStartAddress
}

func (b *Builder) WriteBytes(bs ...byte) {
	b.o = append(b.o, bs...)
}

func (b *Builder) WriteValue(size int, value uint32) {
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf, value)
	b.WriteBytes(buf...)
}

func (o *Builder) Build(textSection []byte, bssSize uint32) []byte {
	textSize := uint32(len(textSection))
	// Size of ELF header + 2 * size program header?
	textOffset := uint32(0x40 + (2 * 0x38))

	// Build ELF Header
	o.WriteBytes(0x7f, 0x45, 0x4c, 0x46) // ELF magic value

	o.WriteBytes(0x02) // 64-bit executable
	o.WriteBytes(0x01) // Little endian
	o.WriteBytes(0x01) // ELF version
	o.WriteBytes(0x00) // Target OS ABI
	o.WriteBytes(0x00) // Further specify ABI version

	o.WriteBytes(0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // Unused bytes

	o.WriteBytes(0x02, 0x00)             // Executable type
	o.WriteBytes(0x3e, 0x00)             // x86-64 target architecture
	o.WriteBytes(0x01, 0x00, 0x00, 0x00) // ELF version

	// 64-bit virtual offsets always start at 0x400000?? https://stackoverflow.com/questions/38549972/why-elf-executables-have-a-fixed-load-address
	// This seems to be a convention set in the x86_64 system-v abi: https://refspecs.linuxfoundation.org/elf/x86_64-SysV-psABI.pdf P26
	o.WriteValue(8, virtualStartAddress+textOffset)

	o.WriteBytes(0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // Offset from file to program header
	o.WriteBytes(0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // Start of section header table
	o.WriteBytes(0x00, 0x00, 0x00, 0x00)                         // Flags
	o.WriteBytes(0x40, 0x00)                                     // Size of this header
	o.WriteBytes(0x38, 0x00)                                     // Size of a program header table entry - This should always be the same for 64-bit
	o.WriteBytes(0x02, 0x00)                                     // Length of sections: data and text for now
	o.WriteBytes(0x00, 0x00)                                     // Size of section header, which we aren't using
	o.WriteBytes(0x00, 0x00)                                     // Number of entries section header
	o.WriteBytes(0x00, 0x00)                                     // Index of section header table entry

	// Build Program Header
	// Text Segment
	o.WriteBytes(0x01, 0x00, 0x00, 0x00) // PT_LOAD, loadable segment. Both data and text segment use this.
	o.WriteBytes(0x05, 0x00, 0x00, 0x00) // Flags: 0x4 executable, 0x2 write, 0x1 read
	o.WriteValue(8, 0)                   // textOffset)          // Offset from the beginning of the file. These values depend on how big the header and segment sizes are.
	o.WriteValue(8, virtualStartAddress)
	o.WriteValue(8, virtualStartAddress) // Physical address, irrelavnt on linux.
	o.WriteValue(8, textSize)            // Number of bytes in file image of segment, must be larger than or equal to the size of payload in segment. Should be zero for bss data.
	o.WriteValue(8, textSize)            // Number of bytes in memory image of segment, is not always same size as file image.
	o.WriteValue(8, alignment)

	// Build Program Header
	// Bss Segment
	o.WriteBytes(0x01, 0x00, 0x00, 0x00)    // PT_LOAD, loadable segment. Both data and text segment use this.
	o.WriteBytes(0x07, 0x00, 0x00, 0x00)    // Flags: 0x4 executable, 0x2 write, 0x1 read // TODO: Which flags are which values exactly???
	o.WriteValue(8, 0)                      // Offset address.
	o.WriteValue(8, bssVirtualStartAddress) // Virtual address.
	o.WriteValue(8, bssVirtualStartAddress) // Physical address.
	o.WriteValue(8, 0)                      // Number of bytes in file image.
	o.WriteValue(8, bssSize)                // Number of bytes in memory image.
	o.WriteValue(8, alignment)

	// Output the text segment
	o.WriteBytes(textSection...)
	return o.o
}
