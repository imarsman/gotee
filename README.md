# tee (gotee)
An implementation of tee in Go.

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

* `gotee -h` print usage
* `gotee -a` append to existing files
* `gotee -i` ignore sigint
  * this is an interesting one. The outcome of using this option is that you
      have to kill the process, as sigint (CTL-C) is ignored. As far as I can
      tell I have implemented this correctly. The way that I got this working
      required an added channel wait in the case of the use of the `-i` (ignore)
      flag. I am not sure I handled this right or at least am not sure I
      understand why usng the conditional channel wait was required.
* `gotee -S` do not forward standard input to standard output
* `gotee <file1> <file2>` - write to all files in list

## Noets

The official `tee` waits for stdin even when nothing has been sent to it. I've
added support for waiting on keyboard input for string data. CTL-C works to exit
this. So although this is using input as a string, this is likely not harmful.
The main stdin reader can handle `cat test.jpg | gotee out.jpg`.

This works to mimic `command | gotee -S out.txt`

`command | dd status=none of=out.txt`

-- Ian Marsman
