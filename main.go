package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	x64e "github.com/vishen/go-brainfunk/x64_encoding"
)

type Compiler struct {
	x64 *x64e.Builder

	program []byte

	nextLoopNumber     int
	loopStack          []int
	loopNumberToOffset map[int]int32
}

func NewCompiler(program []byte) *Compiler {
	c := &Compiler{
		program:            program,
		loopNumberToOffset: make(map[int]int32),
		x64:                x64e.NewBuilder(),
	}

	// Some initialisation.
	// Set up the .bss segment to contain the cells and an output
	// variable to write to.
	cells := c.x64.BssAdd(1024 * 64)
	output := c.x64.BssAdd(1024)

	// Add the
	c.x64.EmitMovRegImm(x64e.RAX, cells)  // mov rax, cells ; current position in cells.
	c.x64.EmitMovRegImm(x64e.R13, 0)      // mov r13, 0 ; tmp reg for moving between ce;ls and output.
	c.x64.EmitMovRegImm(x64e.R14, 0)      // mov r14, 0 ; length of output.
	c.x64.EmitMovRegImm(x64e.R15, output) // mov r15, output ; current position in output.
	c.x64.EmitMovRegImm(x64e.R12, output) // mov r12, output ; store the output offset so it is easier to write to stdout.
	return c
}

func (c *Compiler) Build() []byte {

	// Add the write and exit after the generated code
	c.x64.EmitMovRegImm(x64e.RAX, 4) // sys_write
	c.x64.EmitMovRegImm(x64e.RBX, 1) // fd 1: stdout
	c.x64.EmitMovRegReg(x64e.RCX, x64e.R12)
	c.x64.EmitMovRegReg(x64e.RDX, x64e.R14)
	c.x64.EmitInt(0x80)

	c.x64.EmitMovRegImm(x64e.RAX, 1) // sys_exit
	c.x64.EmitMovRegImm(x64e.RBX, 3) // return code // TODO: return 3 for testing.
	c.x64.EmitInt(0x80)
	return c.x64.Build()
}

func (c *Compiler) EmitInc() {
	c.x64.EmitIncMem(x64e.RAX, 0)
}
func (c *Compiler) EmitDec() {
	c.x64.EmitDecMem(x64e.RAX, 0)
}
func (c *Compiler) EmitNext() {
	c.x64.EmitAddRegImm(x64e.RAX, 64)
}
func (c *Compiler) EmitPrev() {
	c.x64.EmitSubRegImm(x64e.RAX, 64)
}
func (c *Compiler) EmitLoop() {
	c.nextLoopNumber += 1
	c.loopStack = append(c.loopStack, c.nextLoopNumber)
	c.loopNumberToOffset[c.nextLoopNumber] = c.x64.CurrentOffset()
}
func (c *Compiler) EmitLoopJump() {
	loopNumber := c.nextLoopNumber
	for i := len(c.loopStack) - 1; i >= 0; i-- {
		if c.loopStack[i] == -1 {
			continue
		}
		loopNumber = c.loopStack[i]
		c.loopStack[i] = -1
		break
	}
	offset := c.loopNumberToOffset[loopNumber]
	c.x64.EmitCmpMemImm(x64e.RAX, 0)
	c.x64.EmitJne(offset)
}
func (c *Compiler) EmitOutputChar() {
	c.x64.EmitMovRegMem(x64e.R13, x64e.RAX, 0)
	c.x64.EmitMovMemReg(x64e.R15, x64e.R13, 0)
	c.x64.EmitIncReg(x64e.R14)
	c.x64.EmitIncReg(x64e.R15)
}

func (c *Compiler) ParseAndEmit() error {
	loopsCounter := 0
	loopsFinished := 0
	for _, ch := range c.program {
		switch ch {
		case '+':
			c.EmitInc()
		case '-':
			c.EmitDec()
		case '>':
			c.EmitNext()
		case '<':
			c.EmitPrev()
		case '.':
			c.EmitOutputChar()
		case '[':
			loopsCounter += 1
			c.EmitLoop()
		case ']':
			loopsFinished += 1
			c.EmitLoopJump()
		}
	}
	if loopsCounter != loopsFinished {
		return fmt.Errorf("unbalanced []: %d opened, %d closed\n", loopsCounter, loopsFinished)
	}
	return nil
}

var (
	outputBinaryName = flag.String("o", "", "binary executable output name. Defaults to the passed in filename")
)

func usage() {
	fmt.Printf("go-brainfunk /path/to/brainfuck-file -o <output-binary>\n")
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
		return
	}

	fileToCompile := flag.Args()[0]
	program, err := ioutil.ReadFile(fileToCompile)
	if err != nil {
		log.Fatalf("unable to open file %q: %v", fileToCompile, err)
	}

	var outputFilename string
	if *outputBinaryName != "" {
		outputFilename = *outputBinaryName
	} else {
		fileBase := filepath.Base(fileToCompile)
		outputFilename = strings.Replace(fileBase, filepath.Ext(fileBase), "", -1)
	}

	comp := NewCompiler(program)
	if err := comp.ParseAndEmit(); err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(outputFilename, comp.Build(), 0755); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote executable to %s\n", outputFilename)
}
