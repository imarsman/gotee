# tee
An implementation of tee in Go

This program, which is by no means original, takes in standard input and for
each file specified either appends or writes to each file until the standard
input is done. As it does so it passes out what is read to standard output.

The `tee` command is a handy way, and one of the only ways to branch standard
input to save to one or more files and to reproduce standard input as standard
output for consumers down the line. The `-a` (append) and `-i` (ignore sigint)
parameters present in the original `tee` are supported here. Additionally, to
allow for the avoidance of redirecting standard output to /dev/null, the `-S`
option allows for the avoidance of carry-over of standard input to standard
output. This is not something I have seen on any other implementations.

## Usage

* `tee -h` print usage
* `tee -a` append to existing files
* `tee -i` ignore sigint
  * this is an interesting one. The outcome of using this option is that you
      have to kill the process, as siging (CTL-C) is ignored. As far as I can
      tell I have implemented this correctly.
* `tee -S` do not forward standard input to standard output
* `tee <file1> <file2>` - write to all files in list


-- Ian Marsman
