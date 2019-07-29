#!/bin/bash

set -e

go run main.go -f examples/hello_world.bf -o brainfunk
gdb brainfunk --command=gdb-commands
