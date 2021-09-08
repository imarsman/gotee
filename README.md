# tee (gotee)
An implementation of tee in Go.

This program, an implementation of the tee command, first available in 1974,
takes in standard input and for each file specified either appends or writes to
each file until the standard input is done. As it does so it passes out what is
read to standard output.

The `tee` command is a handy way, and one of the only ways to branch standard
input to save to one or more files and to reproduce standard input as standard
output for consumers down the line.  Additionally, to allow for the avoidance of
redirecting standard output to /dev/null, the `-S` option allows for the
avoidance of carry-over of standard input to standard output. This is not
something I have seen on any other implementations.

## Usage

* `gotee -h` print usage
* `gotee` no files specified - use keyboard input instead of stdin and exit on
  Control-C
* `gotee <file1> <file2>` - write to one or more files specified as last arguments
* `gotee -a <file> <file>` - append to existing files and if not existing create
  new files
* `gotee -S <file> <file>` - do not forward standard input to standard output
* `gotee -i` - ignore interrupt. Not implemented but is in original tee
  * I don't quite know what to do with this. If an interrupt is received that
    means that whatever is piping to standard input would have been shut down
    and therefore there would be no standard input to recieve. I would be happy
    to be corrected on this. In theory I could make the loop through standard
    input a function internal to main and restart that function on interrupt,
    but I don't see the point of that. What I have done is in all cases
    intercept an interrupt signal and shut things down as gracefully as
    possible.

## Notes

The official `tee` waits for stdin even when nothing has been sent to it. I've
added support for waiting on keyboard input for string data. Control-C works to
exit this. So although this is reading input as a string, this is likely not
harmful. The main stdin reader reads in bytes and thus can handle `cat test.jpg
| gotee -S out.jpg`.

This works to mimic `command | gotee -S out.txt`

`command | dd status=none of=out.txt`

## Notes

The argument parsing library used here does not deal with arguments such as -1,
-2, -, etc. It may be that an argument will need to have a different identifier to
work around this.

-- Ian Marsman
