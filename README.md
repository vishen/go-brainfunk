# Brainfunk

`go-brainfunk` is a brainfuck compiler that emits generated x64 elf for Linux.
The x64 instructions are generated and encoded programatically and the raw bytes
are used to output an elf executable. This is my first time generating x64 encodings
and elf executables manually, so there is likely mistakes and better approaches.

Only a few x64 instructions were required for a brainfuck program:

- mov
- inc
- dec
- add
- sub
- cmp
- jne
- int 0x80

So I only included the x64 encodings for these instructions, and only
the 64-bit version of these instructions.

The elf executable is also generated programatically and will ouput an elf 
executable to disk. The executable is very minimal, it only includes
the required elf header + 2 program headers, one for `.text` segment
and one for `.bss` segment and then the encoded x64 instructions. 
The `.text` segment header contains information about the x64 code, 
and the `.bss` segment contains information about uninitialised data.

The resulting binary is quite small because it is missing all debug
information usually produced by compilers and linkers.

The compiler isn't very smart, it doesn't attempt to do any constant
folding or correct back-patching for uninitialized data. The elf binary
will always have the `.text` section start from `0x400000` and the 
unitialised data section always starts from `0x600000`. Since this is
always the case we can "hardcode" the uninitialised data addresses.

## x86-64 Instruction Encoding

- REX (0-1 bytes) | Opcode (1-3 bytes) | MODR/M (0 -1 bytes) | SIB (0-1 bytes) | Displacement (0, 1, 2 or 4 bytes) | Immediate (0, 1, 2 or 4 bytes)

### REX

The REX prefix is only available in *long mode*. An REX prefix
must be encoded when:

- using 64-bit operand size and the instruction does not default to 64-bit operand size
- using one of the extended registers (R8 to R15, XMM8 to XMM15, YMM8 to YMM15, CR8 to CR15 and DR8 to DR15)
- using one of the uniform byte registers SPL, BPL, SIL or DIL

A REX prefix must not be encoded when:

- using one of the high byte registers AH, CH, BH or DH.

#### REX Encoding

| 7 | 6 | 5 | 4 | 3 | 2 | 1 | 0 |
| 0   1   0   0 | W | R | X | B |

- 0100 is a 4 bit fixed bit pattern.
- W (1 bit): 1 when a 64-bit operand size is used. Otherwise, 0 will use the default 32-bit operand size.
- R (1 bit): Extension of MODRM.reg field.
- X (1 bit): Extension of SIB.index field.
- B (1 bit): Entension of MODRM.rm or SIB.base field.

### MOD R/M

Used to encode up to two operands of an instruction, each of which is a 
direct register or effective memory address.

#### MOD R/M Encoding

| 7 | 6 | 5 | 4 | 3 | 2 | 1 | 0 |
|  mod  |    reg    |    rm     |

- MODRM.mod (2 bits):
	- 00 -> [rax]
	- 01 -> [rax + imm8], an immediate / constant 8 bit value
	- 10 -> [rax + imm32], an immediate / constant 32 bit value
	- 11 -> rax
- MODRM.reg (3 bits): 
	- Opcode extension: used by some instructions but has no further meaning other than distinguishing the instruction from other instructions.
	- Register reference: can be used as the source or destination of an instruction.
- MODRM.rm (3 bits): Specifies a direct or indirect register operand, optionally with a displacement.

## Elf Executable

The elf executable is consistent of 4 parts all layed out one after the other
in the executable: elf header, text program header, bss program header and
the raw x64 encodings. 

The generated elf executable is currently missing debug information, so 
tools like `gdb` and `objdump` don't work on the resulting binaries. However,
`gdb` can be told to work without the debug information present.

Currently, the elf generation always assumes the brainfuck program can
fit into 0x200000 bytes of memory. If a proram goes over this I believe
there will be segfaults and all other sorts of issues.

### Program headers

These are used to indicate where abouts in the executable binary a
certain sections starts, either the `.text` or `.bss` segment for
this compiler. Most of the fields are pretty self explanatory, but 
the virtual address is the virtual address in memory you want linux
to allocate you memory, this is usually started at 0x400000 by 
convention _I think_. The physical address apparently doesn't matter
for my use case, but other elf executables I checked, like `protoc` 
and `go` have the physical address set to the virtual address. The 
number of bytes in the file image is how many bytes is in the segment,
so for `.text` it will be the length of the encoded data, for the
`.bss` segment it will be zero since nothing is added in the resulting 
binary executable for a `.bss` segment. And the number of bytes in the 
memory image is how much memory it will take up once the program is loaded;
this will be the size requested by the program and will be filled out
with zeroed out data when the program is loaded into memory.
that segment takes up

## Resources

- https://gist.github.com/mikesmullin/6259449
- https://www.systutorials.com/72643/beginners-guide-x86-64-instruction-encoding/
- https://wiki.osdev.org/X86-64_Instruction_Encoding
- https://www.felixcloutier.com/x86/
- http://www.c-jump.com/CIS77/CPU/x86/lecture.html
- https://github.com/TinyCC/tinycc/blob/dev/x86_64-gen.c
- https://github.com/TinyCC/tinycc/blob/dev/x86_64-asm.h
