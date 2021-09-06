package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/jwalton/gchalk"
)

const (
	brightGreen = iota
	brightYellow
	brightBlue
	brightRed
	noColour // Can use to default to no colour output
)

// This package is designed to allow for easy intake of standard input and
// logical writing of the contents of standard intput to one or more files
// specified by the invocation of this command. Most of the logic happens at the
// end of the main method. The reading of standard input (not from keyboard) is
// an iterative process, so writes can be done for each file in sequence. If
// there turn out to be concurrency issues channels can be used or some other
// mechanism.

var useColour = true // use colour - defaults to true
var c chan (os.Signal)

// Used to prevent exit on siging with -i option
var doneChannel = make(chan bool)

var readWriter *bufio.ReadWriter
var fileContainer *container
var exitStatus = 0

func init() {
	c = make(chan os.Signal, 1)
	fileContainer = newContainer()

	br := bufio.NewReader(os.Stdin)
	bw := bufio.NewWriter(os.Stdout)

	readWriter = bufio.NewReadWriter(br, bw)
}

// fileWriter struct to help manage writing to a file
type fileWriter struct {
	file   *os.File
	writer *bufio.Writer
	active bool
}

// newFileWriter properly initialize a new fileWriter, including catching errors
func newFileWriter(path string, append bool) (writer *fileWriter, err error) {

	writer = new(fileWriter)
	mode := os.O_APPEND
	if append == false {
		mode = os.O_CREATE
	}
	if _, err = os.Stat(path); err != nil {
		mode = os.O_CREATE
		writer.file, err = os.Create(path)
		if err != nil {
			// Something wrong like bad file path
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	} else {
		if append == false {
			writer.file, err = os.Create(path)
			if err != nil {
				// Something wrong like bad file path
				fmt.Fprintln(os.Stderr, err.Error())
				return
			}
		}
	}

	writer.active = true
	writer.file, err = os.OpenFile(path, mode|os.O_WRONLY, 0644)
	if err != nil {
		// Something wrong like bad file path
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, err
	}
	writer.writer = bufio.NewWriter(writer.file)

	return
}

// write write bytes to the bufio.Writer
func (s *fileWriter) write(bytes []byte) error {
	if _, err := s.writer.Write(bytes); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// close close the underlying writer
func (s *fileWriter) close() {
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	s.file.Close()
}

// container holds slice of fileWriters
type container struct {
	fileWriters []*fileWriter
}

// newContainer properly initialize a new container
func newContainer() *container {
	c := new(container)
	c.fileWriters = make([]*fileWriter, 0, 5)

	return c
}

// addFileWriter add a fileWriter to the container's slice
func (c *container) addFileWriter(path string, appendToFile bool) (writer *fileWriter, err error) {
	writer, err = newFileWriter(path, appendToFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Probem obtaining fileWriter for pth", path)
		return nil, err
	}
	c.fileWriters = append(c.fileWriters, writer)

	return
}

// write incoming bytes to all fileWriters
func (c *container) write(bytes []byte) {
	fmt.Println("got", string(bytes))
	for _, s := range c.fileWriters {
		s.write(bytes)
	}
}

// close call close on all fileWriters
func (c *container) close() {
	for _, s := range c.fileWriters {
		s.close()
	}
}

func colour(colour int, input ...string) (output string) {
	str := fmt.Sprint(strings.Join(input, " "))
	str = strings.Replace(str, "  ", " ", -1)

	output = str
	if !useColour {
		return
	}

	// Choose colour for output or none
	switch colour {
	case brightGreen:
		output = gchalk.BrightGreen(str)
	case brightYellow:
		output = gchalk.BrightYellow(str)
	case brightBlue:
		output = gchalk.BrightBlue(str)
	case brightRed:
		output = gchalk.BrightRed(str)
	}

	return
}

// printHelp print out simple help output
func printHelp(out *os.File) {
	fmt.Fprintln(out, colour(brightGreen, os.Args[0], "- a simple tee program"))
	fmt.Fprintln(out, "Usage")
	fmt.Fprintln(out, "Takes standard input, saves it to files, and repeats to stdout")
	fmt.Fprintln(out, "Example: tee -i -a file1.txt file2.txt")

	// Prints to stdout
	flag.PrintDefaults()
	os.Exit(0)
}

func main() {

	var helpFlag bool
	flag.BoolVar(&helpFlag, "h", false, "print usage")

	// Colour output only written to stderr. Otherwise verbatim input to output.
	// var noColourFlag bool
	// flag.BoolVar(&noColourFlag, "C", false, "no colour output")
	// useColour = !noColourFlag

	var stdoutFlag bool
	flag.BoolVar(&stdoutFlag, "S", false, "do not forward standard input to standard output")

	var appendFlag bool
	flag.BoolVar(&appendFlag, "a", false, "append to files if they already exist")

	flag.Parse()

	stdoutFlag = !stdoutFlag

	// args are interpreted as paths
	args := flag.Args()

	if helpFlag {
		out := os.Stderr
		printHelp(out)
	}

	if len(args) == 0 {
		out := os.Stderr
		fmt.Fprintln(out, colour(brightRed, "No files specified. Exiting with usage information."))
		printHelp(out)
	}

	// Handle interrupt (Control-C) signal. Doing this to do cleanup. The -i flag in the
	// official tee stops interrupt from working. I don't know what this does
	// beyond letting final write happen, which the below goroutine does in all cases.
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// Block writing to stdErr
			// There may be a better way to allow sig to be defined
			stdErr := os.Stderr
			os.Stderr = nil
			fmt.Fprintln(os.Stderr, colour(brightRed, "got signal", sig.String()))
			readWriter.Writer.Flush()
			for _, s := range fileContainer.fileWriters {
				s.close()
			}
			os.Stderr = stdErr

			exitStatus = 1
			os.Exit(exitStatus)
		}
	}()

	// Use stdin if available, otherwise exit, as stdin is what this is all about.
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
	} else {
		// container := newContainer()
		// Wait on keyboard input. Exit with Control-C.
		// Iterate through file path args to make file writers
		for i := 0; i < len(args); i++ {
			if strings.Contains(args[i], "*") {
				exitStatus = 1
				continue
			}
			_, err := fileContainer.addFileWriter(args[i], appendFlag)
			if err != nil {
				exitStatus = 1
				continue
			}
		}
		for {
			// Read new line of input
			input, isPrefix, err := readWriter.ReadLine()
			if err != nil && err != io.EOF {
				exitStatus = 1
				os.Exit(exitStatus)
			}

			if isPrefix {
				fmt.Fprintln(os.Stderr, "line too long")
			}
			// Write line of input to all fileWriters
			for i := 0; i < len(fileContainer.fileWriters); i++ {
				fileWriter := fileContainer.fileWriters[i]
				if fileWriter.active {
					err := fileWriter.write(
						[]byte(
							fmt.Sprintf(
								"%s\n",
								string(input),
							)))
					fileWriter.writer.Flush()
					if err != nil {
						exitStatus = 1
						fileWriter.active = false
					}
				}
			}
		}
	}

	// Iterate through file path args
	for i := 0; i < len(args); i++ {
		if strings.Contains(args[i], "*") {
			exitStatus = 1
			continue
		}
		_, err := fileContainer.addFileWriter(args[i], appendFlag)
		if err != nil {
			exitStatus = 1
			continue
		}
	}
	if len(fileContainer.fileWriters) == 0 {
		fmt.Fprintln(os.Stderr, colour(brightRed, "no files to write to"))
		exitStatus = 1
		os.Exit(exitStatus)
	}

	buf := make([]byte, 2048)
	count := 0
	// eof := false // eof indicates actual ending of input (plus err.EOF)
	for {
		n, err := readWriter.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err.Error())
			break
		}
		if n == 0 && err == io.EOF {
			break
		}
		// Send bytes to each file fileWriter
		for i := 0; i < len(fileContainer.fileWriters); i++ {
			fileWriter := fileContainer.fileWriters[i]
			if fileWriter.active {
				err := fileWriter.write(buf[0:n])
				fileWriter.writer.Flush()
				if err != nil {
					fileWriter.active = false
				}
			}
		}
		if stdoutFlag {
			readWriter.Write(buf[0:n])
			// The write method for fileWriter.write does flush.
			readWriter.Flush()
		}
		count++
	}

	os.Exit(exitStatus)
}
