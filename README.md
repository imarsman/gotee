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
  * I don't quite know what to do with this. I have currently implemented this
    flag to detect an interrupt and try to cleanly shut everything down. If an
    interrupt is received that means that whatever is piping to standard input
    would have been shut down and therefore there would be no standard input to
    recieve. I would be happy to be corrected on this. In theory I could make
    the loop through standard input a function internal to main and restart that
    function on interrupt, but I don't see the point of that.
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
