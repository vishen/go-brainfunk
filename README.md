

## x86-64 Instruction Encoding

- REX (0-1 bytes) | Opcode (1-3 bytes) | MODR/M (0 -1 bytes) | SIB (0-1 bytes) | Displacement (0, 1, 2 or 4 bytes) | Immediate (0, 1, 2 or 4 bytes)

## REX

The REX prefix is only available in *long mode*. An REX prefix
must be encoded when:

- using 64-bit operand size and the instruction does not default to 64-bit operand size
- using one of the extended registers (R8 to R15, XMM8 to XMM15, YMM8 to YMM15, CR8 to CR15 and DR8 to DR15)
- using one of the uniform byte registers SPL, BPL, SIL or DIL

A REX prefix must not be encoded when:

- using one of the high byte registers AH, CH, BH or DH.


### Encoding

| 7 | 6 | 5 | 4 | 3 | 2 | 1 | 0 |
| 0   1   0   0 | W | R | X | B |

- 0100 is a 4 bit fixed bit pattern.
- W (1 bit): 1 when a 64-bit operand size is used. Otherwise, 0 will use the default 32-bit operand size.
- R (1 bit): Extension of MODRM.reg field.
- X (1 bit): Extension of SIB.index field.
- B (1 bit): Entension of MODRM.rm or SIB.base field.

## MOD R/M

Used to encode up to two operands of an instruction, each of which is a 
direct register or effective memory address.

### Encoding

| 7 | 6 | 5 | 4 | 3 | 2 | 1 | 0 |
|  mod  |    reg    |    rm     |

- MODRM.mod (2 bits): When b11 (binary 11), the register-direct addressing mode is used, otherwise register-indirect addressing mode is used.
- MODRM.reg (3 bits): 
	- Opcode extension: used by some instructions but has no further meaning other than distinguishing the instruction from other instructions.
	- Register reference: can be used as the source or destination of an instruction.
- MODRM.rm (3 bits): Specifies a direct or indirect register operand, optionally with a displacement.

## Resources

### Output x64

- https://gist.github.com/mikesmullin/6259449
- https://www.systutorials.com/72643/beginners-guide-x86-64-instruction-encoding/
- https://wiki.osdev.org/X86-64_Instruction_Encoding
- https://www.felixcloutier.com/x86/
- http://www.c-jump.com/CIS77/CPU/x86/lecture.html
- https://github.com/TinyCC/tinycc/blob/dev/x86_64-gen.c
- https://github.com/TinyCC/tinycc/blob/dev/x86_64-asm.h
