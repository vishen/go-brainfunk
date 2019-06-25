#!/bin/bash

set -ex

go run main.go > comp.asm
nasm -f elf64 -g comp.asm 
ld -m elf_x86_64 -o comp comp.o

./comp
