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
	loopNumberToAddrID map[int]int

	memoryIndexMax int32
	outputOffset   int32 // Offset in program where sys_write fuction is.
}

func NewCompiler(program []byte) *Compiler {
	c := &Compiler{
		program:            program,
		loopNumberToOffset: make(map[int]int32),
		loopNumberToAddrID: make(map[int]int),
		x64:                x64e.NewBuilder(),
	}

	// Some initialisation.
	// Set up the .bss segment to contain the cells.
	cells := c.x64.BssAdd(1024 * 64) // [1000]int64

	c.x64.EmitJmpForwardRelative(23) // Length of stdout function below

	c.outputOffset = c.x64.CurrentOffset()
	// Add jump to exit above the write, once the write
	// has been made into a function that returns
	c.x64.EmitMovRegReg(x64e.RCX, x64e.RAX)
	c.x64.EmitMovRegImm(x64e.RAX, 4) // sys_write
	c.x64.EmitMovRegImm(x64e.RBX, 1) // fd 1: stdout
	c.x64.EmitMovRegImm(x64e.RDX, 1)
	c.x64.EmitInt(0x80)
	c.x64.EmitRet()

	c.x64.EmitMovRegImm(x64e.RAX, cells) // mov rax, cells ; current position in cells.
	c.x64.EmitMovRegImm(x64e.R15, 0)     // mov r15, 0 ; this is where the character to be outputted will be.

	return c
}

func (c *Compiler) Build() []byte {
	// Add the exit after the generated code.

	c.x64.EmitMovRegImm(x64e.RAX, 1) // sys_exit
	c.x64.EmitMovRegImm(x64e.RBX, 0) // return code
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
	c.memoryIndexMax += 1
}
func (c *Compiler) EmitPrev() {
	c.x64.EmitSubRegImm(x64e.RAX, 64)
	c.memoryIndexMax -= 1
}
func (c *Compiler) EmitLoop() {
	c.nextLoopNumber += 1
	c.loopStack = append(c.loopStack, c.nextLoopNumber)
	c.loopNumberToOffset[c.nextLoopNumber] = c.x64.CurrentOffset()
	c.x64.EmitCmpMemImm(x64e.RAX, 0)
	addrID := c.x64.EmitJeqNotYetDefined()
	c.loopNumberToAddrID[c.nextLoopNumber] = addrID
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
	length := c.x64.EmitJneBack(offset) // TODO: Why is it the length of the jne..?
	c.x64.CompleteJeq(c.loopNumberToAddrID[loopNumber], c.x64.CurrentOffset(), length)
}
func (c *Compiler) EmitOutputChar() {
	c.x64.EmitMovRegReg(x64e.R14, x64e.RAX)
	c.x64.EmitCall(c.outputOffset)
	c.x64.EmitMovRegReg(x64e.RAX, x64e.R14)
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
	inputFilename    = flag.String("f", "", "path to bainfuck program to compile")
	outputBinaryName = flag.String("o", "", "binary executable output name. Defaults to the passed in filename")
)

func usage() {
	fmt.Printf("usage: go-brainfunk -f /path/to/brainfuck-file -o <output-binary>\n")
}

func main() {
	flag.Parse()

	fileToCompile := *inputFilename
	if fileToCompile == "" {
		fmt.Printf("missing required flag -f <path/to/brainfuck program>\n")
		usage()
		return
	}
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
