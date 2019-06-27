package x64_encoding

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestGeneration(t *testing.T) {
	instr := []struct {
		name     string
		f        func(b *Builder)
		expected []byte
	}{
		// https://defuse.ca/online-x86-assembler.htm
		/*
			0:  cd 80                   int    0x80
		*/
		{"int 0x80", func(b *Builder) { b.EmitInt(0x80) }, []byte{0xcd, 0x80}},

		// Mov text
		{"mov rax, 0x01", func(b *Builder) { b.EmitMovRegImm(RAX, 0x01) }, []byte{0x48, 0xc7, 0xc0, 0x01, 0x00, 0x00, 0x00}},
		{"mov r15, 0x15", func(b *Builder) { b.EmitMovRegImm(R15, 0x15) }, []byte{0x49, 0xc7, 0xc7, 0x15, 0x00, 0x00, 0x00}},
		{"mov rax, rbx", func(b *Builder) { b.EmitMovRegReg(RAX, RBX) }, []byte{0x48, 0x89, 0xd8}},
		{"mov rax, r13", func(b *Builder) { b.EmitMovRegReg(RAX, R13) }, []byte{0x4c, 0x89, 0xe8}},
		{"mov r13, rbx", func(b *Builder) { b.EmitMovRegReg(R13, RBX) }, []byte{0x49, 0x89, 0xdd}},
		{"mov r13, r14", func(b *Builder) { b.EmitMovRegReg(R13, R14) }, []byte{0x4d, 0x89, 0xf5}},
		{"mov qword [r13], r14", func(b *Builder) { b.EmitMovMemReg(R13, R14, 0) }, []byte{0x4d, 0x89, 0x75, 0x00}},
		{"mov qword [r13 + 0x04], r14", func(b *Builder) { b.EmitMovMemReg(R13, R14, 0x04) }, []byte{0x4d, 0x89, 0x75, 0x04}},
		{"mov qword [r13 + 0x80], r14", func(b *Builder) { b.EmitMovMemReg(R13, R14, 0x80) }, []byte{0x4d, 0x89, 0xb5, 0x80, 0x00, 0x00, 0x00}},
		{"mov r13, qword [r14]", func(b *Builder) { b.EmitMovRegMem(R13, R14, 0) }, []byte{0x4d, 0x8b, 0x2e}},
		{"mov r13, qword [r14 + 0x05]", func(b *Builder) { b.EmitMovRegMem(R13, R14, 0x05) }, []byte{0x4d, 0x8b, 0x6e, 0x05}},
		{"mov r13, qword [r14 + 0x81]", func(b *Builder) { b.EmitMovRegMem(R13, R14, 0x81) }, []byte{0x4d, 0x8b, 0xae, 0x81, 0x00, 0x00, 0x00}},
		{"mov r13, qword [rax]", func(b *Builder) { b.EmitMovRegMem(R13, RAX, 0x00) }, []byte{0x4c, 0x8b, 0x28}},
		{"mov r13, qword [rbx]", func(b *Builder) { b.EmitMovRegMem(R13, RBX, 0x00) }, []byte{0x4c, 0x8b, 0x2b}},
		{"mov r13, qword [rbx+0x81]", func(b *Builder) { b.EmitMovRegMem(R13, RBX, 0x81) }, []byte{0x4c, 0x8b, 0xab, 0x81, 0x00, 0x00, 0x00}},

		/*
			0:  48 ff c0                inc    rax
			3:  49 ff c6                inc    r14
			6:  49 ff 45 00             inc    QWORD PTR [r13+0x0]
			a:  49 ff 45 04             inc    QWORD PTR [r13+0x4]
			e:  49 ff 85 81 00 00 00    inc    QWORD PTR [r13+0x81]
		*/
		{"inc rax", func(b *Builder) { b.EmitIncReg(RAX) }, []byte{0x48, 0xff, 0xc0}},
		{"inc r14", func(b *Builder) { b.EmitIncReg(R14) }, []byte{0x49, 0xff, 0xc6}},
		{"inc [r13]", func(b *Builder) { b.EmitIncMem(R13, 0) }, []byte{0x49, 0xff, 0x45, 0x00}},
		{"inc [r13+0x04]", func(b *Builder) { b.EmitIncMem(R13, 4) }, []byte{0x49, 0xff, 0x45, 0x04}},
		{"inc [r13+0x81]", func(b *Builder) { b.EmitIncMem(R13, 0x81) }, []byte{0x49, 0xff, 0x85, 0x81, 0x00, 0x00, 0x00}},

		/*
			0:  48 ff c8                dec    rax
			3:  49 ff ce                dec    r14
			6:  49 ff 4d 00             dec    QWORD PTR [r13+0x0]
			a:  49 ff 4d 04             dec    QWORD PTR [r13+0x4]
			e:  49 ff 8d 81 00 00 00    dec    QWORD PTR [r13+0x81]
		*/
		{"dec rax", func(b *Builder) { b.EmitDecReg(RAX) }, []byte{0x48, 0xff, 0xc8}},
		{"dec r14", func(b *Builder) { b.EmitDecReg(R14) }, []byte{0x49, 0xff, 0xce}},
		{"dec [r13]", func(b *Builder) { b.EmitDecMem(R13, 0) }, []byte{0x49, 0xff, 0x4d, 0x00}},
		{"dec [r13] + 0x04", func(b *Builder) { b.EmitDecMem(R13, 4) }, []byte{0x49, 0xff, 0x4d, 0x04}},
		{"dec [r13] + 0x81", func(b *Builder) { b.EmitDecMem(R13, 0x81) }, []byte{0x49, 0xff, 0x8d, 0x81, 0x00, 0x00, 0x00}},

		/*
			0:  48 83 c0 01             add    rax,0x1
			4:  48 05 81 00 00 00       add    rax,0x81
			a:  48 83 c3 01             add    rbx,0x1
			e:  48 81 c3 81 00 00 00    add    rbx,0x81
			15: 49 83 c3 01             add    r11,0x1
			19: 49 81 c3 81 00 00 00    add    r11,0x81
			20: 48 01 c3                add    rbx,rax
			23: 4c 01 db                add    rbx,r11
			26: 49 01 c3                add    r11,rax
			29: 4d 01 e3                add    r11,r12
			2c: 49 03 1b                add    rbx,QWORD PTR [r11]
			2f: 4c 03 1b                add    r11,QWORD PTR [rbx]
			32: 4c 03 5b 04             add    r11,QWORD PTR [rbx+0x4]
			36: 4c 03 9b 81 00 00 00    add    r11,QWORD PTR [rbx+0x81]
			3d: 49 01 00                add    QWORD PTR [r8],rax
			40: 49 01 40 04             add    QWORD PTR [r8+0x4],rax
			44: 49 01 80 81 00 00 00    add    QWORD PTR [r8+0x81],rax
			4b: 49 01 98 81 00 00 00    add    QWORD PTR [r8+0x81],rbx
		*/
		{"add rax, 0x01", func(b *Builder) { b.EmitAddRegImm(RAX, 0x01) }, []byte{0x48, 0x83, 0xc0, 0x01}},
		{"add rax, 0x81", func(b *Builder) { b.EmitAddRegImm(RAX, 0x81) }, []byte{0x48, 0x05, 0x81, 0x00, 0x00, 0x00}},
		{"add r11, 0x01", func(b *Builder) { b.EmitAddRegImm(R11, 0x01) }, []byte{0x49, 0x83, 0xc3, 0x01}},
		{"add r11, 0x81", func(b *Builder) { b.EmitAddRegImm(R11, 0x81) }, []byte{0x49, 0x81, 0xc3, 0x81, 0x00, 0x00, 0x00}},
		{"add rbx, rax", func(b *Builder) { b.EmitAddRegReg(RBX, RAX) }, []byte{0x48, 0x01, 0xc3}},
		{"add rbx, r11", func(b *Builder) { b.EmitAddRegReg(RBX, R11) }, []byte{0x4c, 0x01, 0xdb}},
		{"add r11, rax", func(b *Builder) { b.EmitAddRegReg(R11, RAX) }, []byte{0x49, 0x01, 0xc3}},
		{"add r11, r12", func(b *Builder) { b.EmitAddRegReg(R11, R12) }, []byte{0x4d, 0x01, 0xe3}},
		{"add rbx, qword [r11]", func(b *Builder) { b.EmitAddRegMem(RBX, R11, 0) }, []byte{0x49, 0x03, 0x1b}},
		{"add r11, qword [rbx]", func(b *Builder) { b.EmitAddRegMem(R11, RBX, 0) }, []byte{0x4c, 0x03, 0x1b}},
		{"add r11, qword [rbx + 0x04]", func(b *Builder) { b.EmitAddRegMem(R11, RBX, 4) }, []byte{0x4c, 0x03, 0x5b, 0x04}},
		{"add r11, qword [rbx + 0x81]", func(b *Builder) { b.EmitAddRegMem(R11, RBX, 0x81) }, []byte{0x4c, 0x03, 0x9b, 0x81, 0x00, 0x00, 0x00}},
		{"add qword [r8], rax", func(b *Builder) { b.EmitAddMemReg(R8, RAX, 0) }, []byte{0x49, 0x01, 0x00}},
		{"add qword [r8+0x04], rax", func(b *Builder) { b.EmitAddMemReg(R8, RAX, 0x04) }, []byte{0x49, 0x01, 0x40, 0x04}},
		{"add qword [r8+0x81], rax", func(b *Builder) { b.EmitAddMemReg(R8, RAX, 0x81) }, []byte{0x49, 0x01, 0x80, 0x81, 0x00, 0x00, 0x00}},
		{"add qword [r8+0x81], rbx", func(b *Builder) { b.EmitAddMemReg(R8, RBX, 0x81) }, []byte{0x49, 0x01, 0x98, 0x81, 0x00, 0x00, 0x00}},

		/*
			0:  48 83 e8 01             sub    rax,0x1
			4:  48 2d 81 00 00 00       sub    rax,0x81
			a:  48 83 eb 01             sub    rbx,0x1
			e:  48 81 eb 81 00 00 00    sub    rbx,0x81
			15: 49 83 eb 01             sub    r11,0x1
			19: 49 81 eb 81 00 00 00    sub    r11,0x81
			20: 48 29 c3                sub    rbx,rax
			23: 4c 29 db                sub    rbx,r11
			26: 49 29 c3                sub    r11,rax
			29: 4d 29 e3                sub    r11,r12
			2c: 49 2b 1b                sub    rbx,QWORD PTR [r11]
			2f: 4c 2b 1b                sub    r11,QWORD PTR [rbx]
			32: 4c 2b 5b 04             sub    r11,QWORD PTR [rbx+0x4]
			36: 4c 2b 9b 81 00 00 00    sub    r11,QWORD PTR [rbx+0x81]
			3d: 49 29 00                sub    QWORD PTR [r8],rax
			40: 49 29 40 04             sub    QWORD PTR [r8+0x4],rax
			44: 49 29 80 81 00 00 00    sub    QWORD PTR [r8+0x81],rax
			4b: 49 29 98 81 00 00 00    sub    QWORD PTR [r8+0x81],rbx
		*/
		{"sub rax, 0x01", func(b *Builder) { b.EmitSubRegImm(RAX, 0x01) }, []byte{0x48, 0x83, 0xe8, 0x01}},
		{"sub rax, 0x81", func(b *Builder) { b.EmitSubRegImm(RAX, 0x81) }, []byte{0x48, 0x2d, 0x81, 0x00, 0x00, 0x00}},
		{"sub rbx, 0x01", func(b *Builder) { b.EmitSubRegImm(RBX, 0x01) }, []byte{0x48, 0x83, 0xeb, 0x01}},
		{"sub rbx, 0x81", func(b *Builder) { b.EmitSubRegImm(RBX, 0x81) }, []byte{0x48, 0x81, 0xeb, 0x81, 0x00, 0x00, 0x00}},
		{"sub r11, 0x01", func(b *Builder) { b.EmitSubRegImm(R11, 0x01) }, []byte{0x49, 0x83, 0xeb, 0x01}},
		{"sub r11, 0x81", func(b *Builder) { b.EmitSubRegImm(R11, 0x81) }, []byte{0x49, 0x81, 0xeb, 0x81, 0x00, 0x00, 0x00}},
		{"sub rbx, rax", func(b *Builder) { b.EmitSubRegReg(RBX, RAX) }, []byte{0x48, 0x29, 0xc3}},
		{"sub rbx, r11", func(b *Builder) { b.EmitSubRegReg(RBX, R11) }, []byte{0x4c, 0x29, 0xdb}},
		{"sub r11, rax", func(b *Builder) { b.EmitSubRegReg(R11, RAX) }, []byte{0x49, 0x29, 0xc3}},
		{"sub r11, r12", func(b *Builder) { b.EmitSubRegReg(R11, R12) }, []byte{0x4d, 0x29, 0xe3}},
		{"sub rbx, qword [r11]", func(b *Builder) { b.EmitSubRegMem(RBX, R11, 0) }, []byte{0x49, 0x2b, 0x1b}},
		{"sub r11, qword [rbx]", func(b *Builder) { b.EmitSubRegMem(R11, RBX, 0) }, []byte{0x4c, 0x2b, 0x1b}},
		{"sub r11, qword [rbx + 0x04]", func(b *Builder) { b.EmitSubRegMem(R11, RBX, 4) }, []byte{0x4c, 0x2b, 0x5b, 0x04}},
		{"sub r11, qword [rbx + 0x81]", func(b *Builder) { b.EmitSubRegMem(R11, RBX, 0x81) }, []byte{0x4c, 0x2b, 0x9b, 0x81, 0x00, 0x00, 0x00}},
		{"sub qword [r8], rax", func(b *Builder) { b.EmitSubMemReg(R8, RAX, 0) }, []byte{0x49, 0x29, 0x00}},
		{"sub qword [r8+0x04], rax", func(b *Builder) { b.EmitSubMemReg(R8, RAX, 0x04) }, []byte{0x49, 0x29, 0x40, 0x04}},
		{"sub qword [r8+0x81], rax", func(b *Builder) { b.EmitSubMemReg(R8, RAX, 0x81) }, []byte{0x49, 0x29, 0x80, 0x81, 0x00, 0x00, 0x00}},
		{"sub qword [r8+0x81], rbx", func(b *Builder) { b.EmitSubMemReg(R8, RBX, 0x81) }, []byte{0x49, 0x29, 0x98, 0x81, 0x00, 0x00, 0x00}},

		/*
			0:  48 83 f8 01             cmp    rax,0x1
			4:  48 3d 81 00 00 00       cmp    rax,0x81
			a:  48 83 fb 01             cmp    rbx,0x1
			e:  48 81 fb 81 00 00 00    cmp    rbx,0x81
			15: 49 83 fb 01             cmp    r11,0x1
			19: 49 81 fb 81 00 00 00    cmp    r11,0x81
			20: 48 39 c3                cmp    rbx,rax
			23: 4c 39 db                cmp    rbx,r11
			26: 49 39 c3                cmp    r11,rax
			29: 4d 39 e3                cmp    r11,r12
			2c: 49 3b 1b                cmp    rbx,QWORD PTR [r11]
			2f: 4c 3b 1b                cmp    r11,QWORD PTR [rbx]
			32: 4c 3b 5b 04             cmp    r11,QWORD PTR [rbx+0x4]
			36: 4c 3b 9b 81 00 00 00    cmp    r11,QWORD PTR [rbx+0x81]
			3d: 49 39 00                cmp    QWORD PTR [r8],rax
			40: 49 39 40 04             cmp    QWORD PTR [r8+0x4],rax
			44: 49 39 80 81 00 00 00    cmp    QWORD PTR [r8+0x81],rax
			4b: 49 39 98 81 00 00 00    cmp    QWORD PTR [r8+0x81],rbx
		*/
		{"cmp rax, 0x01", func(b *Builder) { b.EmitCmpRegImm(RAX, 0x01) }, []byte{0x48, 0x83, 0xf8, 0x01}},
		{"cmp rax, 0x81", func(b *Builder) { b.EmitCmpRegImm(RAX, 0x81) }, []byte{0x48, 0x3d, 0x81, 0x00, 0x00, 0x00}},
		{"cmp rbx, 0x01", func(b *Builder) { b.EmitCmpRegImm(RBX, 0x01) }, []byte{0x48, 0x83, 0xfb, 0x01}},
		{"cmp rbx, 0x81", func(b *Builder) { b.EmitCmpRegImm(RBX, 0x81) }, []byte{0x48, 0x81, 0xfb, 0x81, 0x00, 0x00, 0x00}},
		{"cmp r11, 0x01", func(b *Builder) { b.EmitCmpRegImm(R11, 0x01) }, []byte{0x49, 0x83, 0xfb, 0x01}},
		{"cmp r11, 0x81", func(b *Builder) { b.EmitCmpRegImm(R11, 0x81) }, []byte{0x49, 0x81, 0xfb, 0x81, 0x00, 0x00, 0x00}},
		{"cmp rbx, rax", func(b *Builder) { b.EmitCmpRegReg(RBX, RAX) }, []byte{0x48, 0x39, 0xc3}},
		{"cmp rbx, r11", func(b *Builder) { b.EmitCmpRegReg(RBX, R11) }, []byte{0x4c, 0x39, 0xdb}},
		{"cmp r11, rax", func(b *Builder) { b.EmitCmpRegReg(R11, RAX) }, []byte{0x49, 0x39, 0xc3}},
		{"cmp r11, r12", func(b *Builder) { b.EmitCmpRegReg(R11, R12) }, []byte{0x4d, 0x39, 0xe3}},
		{"cmp rbx, qword [r11]", func(b *Builder) { b.EmitCmpRegMem(RBX, R11, 0) }, []byte{0x49, 0x3b, 0x1b}},
		{"cmp r11, qword [rbx]", func(b *Builder) { b.EmitCmpRegMem(R11, RBX, 0) }, []byte{0x4c, 0x3b, 0x1b}},
		{"cmp r11, qword [rbx + 0x04]", func(b *Builder) { b.EmitCmpRegMem(R11, RBX, 4) }, []byte{0x4c, 0x3b, 0x5b, 0x04}},
		{"cmp r11, qword [rbx + 0x81]", func(b *Builder) { b.EmitCmpRegMem(R11, RBX, 0x81) }, []byte{0x4c, 0x3b, 0x9b, 0x81, 0x00, 0x00, 0x00}},
		{"cmp qword [r8], rax", func(b *Builder) { b.EmitCmpMemReg(R8, RAX, 0) }, []byte{0x49, 0x39, 0x00}},
		{"cmp qword [r8+0x04], rax", func(b *Builder) { b.EmitCmpMemReg(R8, RAX, 0x04) }, []byte{0x49, 0x39, 0x40, 0x04}},
		{"cmp qword [r8+0x81], rax", func(b *Builder) { b.EmitCmpMemReg(R8, RAX, 0x81) }, []byte{0x49, 0x39, 0x80, 0x81, 0x00, 0x00, 0x00}},
		{"cmp qword [r8+0x81], rbx", func(b *Builder) { b.EmitCmpMemReg(R8, RBX, 0x81) }, []byte{0x49, 0x39, 0x98, 0x81, 0x00, 0x00, 0x00}},
		{"cmp qword [rax], 0x00", func(b *Builder) { b.EmitCmpMemImm(RAX, 0) }, []byte{0x48, 0x83, 0x38, 0x00}},
	}

	for _, ins := range instr {
		t.Run(ins.name, func(t *testing.T) {
			b := &Builder{}
			ins.f(b)
			if !bytes.Equal(b.output, ins.expected) {
				t.Errorf("unexpected generated output %s, expected %s", hexB(b.output), hexB(ins.expected))
			}
		})
	}
}

func TestJne(t *testing.T) {
	/*
		0:  48 c7 c0 00 00 00 00    mov    rax,0x0
		0000000000000007 <loop1>:
		7:  48 83 c0 02             add    rax,0x2
		b:  48 83 f8 0a             cmp    rax,0xa
		f:  75 f6                   jne    7 <loop1>
	*/
	b := &Builder{}
	b.EmitMovRegImm(RAX, 0x01) // mov rax, 0x00
	offset := len(b.output)    // loop1:
	b.EmitAddRegImm(RAX, 0x02) // add rax, 0x02
	b.EmitCmpRegImm(RAX, 10)   // cmp rax, 0xa
	b.EmitJne(int32(offset))   // jne loop1

	expectedOutput := []byte{
		0x48, 0xc7, 0xc0, 0x01, 0x00, 0x00, 0x00, // mov rax, 0x00
		// loop1:
		0x48, 0x83, 0xc0, 0x02, // add rax, 0x02
		0x48, 0x83, 0xf8, 0x0a, // cmp rax, 10
		0x75, 0xf6, // jne loop1
	}
	if !bytes.Equal(b.output, expectedOutput) {
		t.Errorf("unexpected generated output %s, expected %s", b.hex(), hexB(expectedOutput))
	}
}

func TestJneLong(t *testing.T) {
	/*
		0:  48 c7 c0 01 00 00 00    mov    rax,0x1
		7:  48 c7 c3 02 00 00 00    mov    rbx,0x2
		000000000000000e <loop1>:
		e:  48 c7 c0 81 00 00 00    mov    rax,0x81
		15: 48 c7 c0 81 00 00 00    mov    rax,0x81
		1c: 48 c7 c0 81 00 00 00    mov    rax,0x81
		23: 48 c7 c0 81 00 00 00    mov    rax,0x81
		2a: 48 c7 c0 81 00 00 00    mov    rax,0x81
		31: 48 c7 c0 81 00 00 00    mov    rax,0x81
		38: 48 c7 c0 81 00 00 00    mov    rax,0x81
		3f: 48 c7 c0 81 00 00 00    mov    rax,0x81
		46: 48 c7 c0 81 00 00 00    mov    rax,0x81
		4d: 48 c7 c0 81 00 00 00    mov    rax,0x81
		54: 48 c7 c0 81 00 00 00    mov    rax,0x81
		5b: 48 c7 c0 81 00 00 00    mov    rax,0x81
		62: 48 c7 c0 81 00 00 00    mov    rax,0x81
		69: 48 c7 c0 81 00 00 00    mov    rax,0x81
		70: 48 c7 c0 81 00 00 00    mov    rax,0x81
		77: 48 c7 c0 81 00 00 00    mov    rax,0x81
		7e: 48 c7 c0 81 00 00 00    mov    rax,0x81
		85: 48 c7 c0 81 00 00 00    mov    rax,0x81
		8c: 48 c7 c0 81 00 00 00    mov    rax,0x81
		93: 48 c7 c0 81 00 00 00    mov    rax,0x81
		9a: 48 c7 c0 81 00 00 00    mov    rax,0x81
		a1: 48 c7 c0 81 00 00 00    mov    rax,0x81
		a8: 48 c7 c0 81 00 00 00    mov    rax,0x81
		af: 48 c7 c0 81 00 00 00    mov    rax,0x81
		b6: 48 c7 c0 81 00 00 00    mov    rax,0x81
		bd: 48 c7 c0 81 00 00 00    mov    rax,0x81
		c4: 48 c7 c0 81 00 00 00    mov    rax,0x81
		cb: 48 c7 c0 81 00 00 00    mov    rax,0x81
		d2: 48 c7 c0 81 00 00 00    mov    rax,0x81
		d9: 48 c7 c0 81 00 00 00    mov    rax,0x81
		e0: 48 c7 c0 81 00 00 00    mov    rax,0x81
		e7: 48 c7 c0 81 00 00 00    mov    rax,0x81
		ee: 48 c7 c0 81 00 00 00    mov    rax,0x81
		f5: 48 c7 c0 81 00 00 00    mov    rax,0x81
		fc: 0f 85 0c ff ff ff       jne    e <loop1>
	*/
	b := &Builder{}
	b.EmitMovRegImm(RAX, 0x01)
	b.EmitMovRegImm(RBX, 0x02)
	offset := b.CurrentOffset() // loop1:
	for i := 0; i < 34; i++ {
		b.EmitMovRegImm(RAX, 0x81)
	}
	b.EmitJne(offset) // jne loop1

	expectedOutput := []byte{
		0x48, 0xc7, 0xc0, 0x01, 0x00, 0x00, 0x00,
		0x48, 0xc7, 0xc3, 0x02, 0x00, 0x00, 0x00,
	}
	// loop1:
	for i := 0; i < 34; i++ {
		expectedOutput = append(
			expectedOutput, 0x48, 0xc7, 0xc0, 0x81, 0x00, 0x00, 0x00,
		)
	}
	// jne loop1
	expectedOutput = append(expectedOutput, 0x0f, 0x85, 0x0c, 0xff, 0xff, 0xff)
	if !bytes.Equal(b.output, expectedOutput) {
		t.Errorf("unexpected generated output %s, expected %s", b.hex(), hexB(expectedOutput))
	}
}

func hexB(b []byte) string {
	return hex.EncodeToString(b)
}

func builderOutput(f func(b *Builder)) []byte {
	b := &Builder{}
	f(b)
	return b.output
}
