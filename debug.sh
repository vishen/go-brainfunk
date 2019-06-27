#!/bin/bash

set -e

#nasm -f elf64 -g templ.asm 
#ld -m elf_x86_64 -o templ templ.o

go run main.go 
gdb brainfunk --command=gdb-commands
