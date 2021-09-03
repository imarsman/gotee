# tee
An implementation of tee in Go

This program, which is by no means original, takes in standard input and for
each file specified either appends or writes to each file until the standard
input is done. As it does so it passes out what is read to standard output.

## Usage

* `tee -h` print usage
* `tee -a` append to existing files
* `tee -i` ignore sigint
* `tee -S` do not forward standard input to standard output
* `tee <file1> <file2>` - write to all files in list
