package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	x64e "github.com/vishen/go-brainfunk/x64_encoding"
)

const testProgram3 = `++++++++[>++++[>++>+++>+++>+<<<<-]>+>+>->>+[<]<-]>>.>---.+++++++..+++.>>.<-.<.+++.------.--------.>>+.>++.`

const testProgram2 = `++>+++++[<+>-]++++++++[<++++++>-]<.`

const testProgram = `>++++[>++++++<-]>-[[<+++++>>+<-]>-]<<[<]>>>>--.<<<-.>>>-.<.<.>---.<<+++.>>>++.<<---.[>]<<.`

type CompiledOutput struct {
	bufASM strings.Builder
	x64    *x64e.Builder

	nextLoopNumber     int
	loopStack          []int
	loopNumberToOffset map[int]int32
}

func NewCompiledOutput() *CompiledOutput {
	c := &CompiledOutput{
		loopNumberToOffset: make(map[int]int32),
		x64:                x64e.NewBuilder(),
	}

	// Some initialisation.
	// TODO: Move somewhere better
	cells := c.x64.BssAdd(1024 * 64)
	output := c.x64.BssAdd(1024)
	c.x64.EmitMovRegImm(x64e.RAX, cells)
	c.x64.EmitMovRegImm(x64e.R13, 0)
	c.x64.EmitMovRegImm(x64e.R14, 0)
	c.x64.EmitMovRegImm(x64e.R15, output)
	c.x64.EmitMovRegImm(x64e.R12, output)
	return c
}

func (c *CompiledOutput) Build() string {

	// Add the write and exit after the generated code
	// TODO: Move to somewhere better! This is a confusing
	// place to put it I think.
	c.x64.EmitMovRegImm(x64e.RAX, 4) // sys_write
	c.x64.EmitMovRegImm(x64e.RBX, 1) // fd 1: stdout
	c.x64.EmitMovRegReg(x64e.RCX, x64e.R12)
	c.x64.EmitMovRegReg(x64e.RDX, x64e.R14)
	c.x64.EmitInt(0x80)

	c.x64.EmitMovRegImm(x64e.RAX, 1) // sys_exit
	c.x64.EmitMovRegImm(x64e.RBX, 3) // return code // TODO: return 3 for testing.
	c.x64.EmitInt(0x80)

	return fmt.Sprintf(`section .bss

buflen equ 1024
cells: resb buflen * 64
output: resb buflen

section .text

global _start
_start:

; initial setup
mov rax, cells	; current position in cells
mov r13, 0		; temp reg for moving between cells and output
mov r14, 0		; length of output
mov r15, output
%s

write:
mov rax, 4 ; sys_write
mov rbx, 1 ; fd 1 stdout
mov rcx, output
mov rdx, r14
int 80h

exit:
mov eax, 1 ; sys_exit call
mov ebx, 0 ; return code
int 80h

`, c.bufASM.String())
}

func (c *CompiledOutput) EmitInc() {
	c.x64.EmitIncMem(x64e.RAX, 0)
	c.bufASM.WriteString("inc qword [rax]\n")
}
func (c *CompiledOutput) EmitDec() {
	c.x64.EmitDecMem(x64e.RAX, 0)
	c.bufASM.WriteString("dec qword [rax]\n")
}
func (c *CompiledOutput) EmitNext() {
	c.x64.EmitAddRegImm(x64e.RAX, 64)
	c.bufASM.WriteString("add rax, 64\n")
}
func (c *CompiledOutput) EmitPrev() {
	c.x64.EmitSubRegImm(x64e.RAX, 64)
	c.bufASM.WriteString("sub rax, 64\n")
}
func (c *CompiledOutput) EmitLoop() {
	c.nextLoopNumber += 1
	loopLabel := fmt.Sprintf("\nloop%d:", c.nextLoopNumber)
	c.bufASM.WriteString(loopLabel + "\n")
	c.loopStack = append(c.loopStack, c.nextLoopNumber)
	c.loopNumberToOffset[c.nextLoopNumber] = c.x64.CurrentOffset()
	fmt.Printf("Loop Offsets: %v\n", c.loopNumberToOffset)
	fmt.Println(c.x64.Hex())
}
func (c *CompiledOutput) EmitLoopJump() {
	loopNumber := c.nextLoopNumber
	for i := len(c.loopStack) - 1; i >= 0; i-- {
		if c.loopStack[i] == -1 {
			continue
		}
		loopNumber = c.loopStack[i]
		c.loopStack[i] = -1
		break
	}
	loopLabel := fmt.Sprintf("loop%d", loopNumber)
	c.bufASM.WriteString("cmp qword [rax], 0\n")
	c.bufASM.WriteString("jne " + loopLabel + "\n\n")

	offset := c.loopNumberToOffset[loopNumber]
	c.x64.EmitCmpMemImm(x64e.RAX, 0)
	c.x64.EmitJne(offset)
}
func (c *CompiledOutput) EmitOutputChar() {
	c.bufASM.WriteString("mov r13, [rax]\n")
	c.bufASM.WriteString("mov [r15], r13\n")
	c.bufASM.WriteString("inc r14\n")
	c.bufASM.WriteString("inc r15\n")

	c.x64.EmitMovRegMem(x64e.R13, x64e.RAX, 0)
	c.x64.EmitMovMemReg(x64e.R15, x64e.R13, 0)
	c.x64.EmitIncReg(x64e.R14)
	c.x64.EmitIncReg(x64e.R15)
}

func ParseAndEmit(output *CompiledOutput, program string) error {
	loopsCounter := 0
	loopsFinished := 0
	for _, ch := range program {
		switch ch {
		case '+':
			output.EmitInc()
		case '-':
			output.EmitDec()
		case '>':
			output.EmitNext()
		case '<':
			output.EmitPrev()
		case '.':
			output.EmitOutputChar()
		case '[':
			loopsCounter += 1
			output.EmitLoop()
		case ']':
			loopsFinished += 1
			output.EmitLoopJump()
		}
	}
	if loopsCounter != loopsFinished {
		return fmt.Errorf("unbalanced []: %d opened, %d closed\n", loopsCounter, loopsFinished)
	}
	return nil
}

func main() {
	output := NewCompiledOutput()
	if err := ParseAndEmit(output, testProgram); err != nil {
		log.Fatal(err)
	}
	compiled := output.Build()
	// fmt.Println(compiled)
	fmt.Println(output.x64.Hex())

	filename := "brainfunk.asm"
	if err := ioutil.WriteFile(filename, []byte(compiled), 0755); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote asm to %s\n", filename)

	filename = "brainfunk"
	data := output.x64.Build()
	if err := ioutil.WriteFile(filename, data, 0755); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote binary to %s\n", filename)
}
