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

var useColour = true // use colour - defaults to true
var c chan (os.Signal)

func init() {
	c = make(chan os.Signal, 1)
}

// Implement -i flag - ignore sigint
func ignoreSignal() {
	// I think os.Interupt will handle sigint
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Fprintln(os.Stderr, colour(brightRed, "ignoring", sig.String()))
		}
	}()
}

// fileWriter struct to help manage writing to a file
type fileWriter struct {
	file   *os.File
	writer *bufio.Writer
	active bool
}

// newFileWriter properly initialize a new fileWriter, including catching errors
func newFileWriter(path string, append bool) (*fileWriter, error) {
	s := new(fileWriter)

	var err error
	mode := os.O_APPEND
	if append == false {
		mode = os.O_CREATE
	}
	if _, err = os.Stat(path); err != nil {
		mode = os.O_CREATE
		s.file, err = os.Create(path)
		if err != nil {
			// Something wrong like bad file path
			fmt.Fprintln(os.Stderr, err.Error())
			return nil, err
		}
	} else {
		if append == false {
			s.file, err = os.Create(path)
			if err != nil {
				// Something wrong like bad file path
				fmt.Fprintln(os.Stderr, err.Error())
				return nil, err
			}
		}
	}

	s.active = true
	s.file, err = os.OpenFile(path, mode|os.O_WRONLY, 0644)
	if err != nil {
		// Something wrong like bad file path
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, err
	}
	s.writer = bufio.NewWriter(s.file)

	return s, nil
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
}

// container holds slice of fileWriters
type container struct {
	writers []*fileWriter
}

// newContainer properly initialize a new container
func newContainer() *container {
	c := new(container)
	c.writers = make([]*fileWriter, 0, 5)

	return c
}

// addFileWriter add a fileWriter to the container's slice
func (c *container) addFileWriter(path string, appendToFile bool) (*fileWriter, error) {
	fileWriter, err := newFileWriter(path, appendToFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Probem obtaining fileWriter for pth", path)
		return nil, err
	}
	c.writers = append(c.writers, fileWriter)

	return fileWriter, nil
}

// write incoming bytes to all fileWriters
func (c *container) write(bytes []byte) {
	for _, s := range c.writers {
		s.write(bytes)
	}
}

// close call close on all fileWriters
func (c *container) close() {
	for _, s := range c.writers {
		s.close()
	}
}

func colour(colour int, input ...string) string {
	str := fmt.Sprint(strings.Join(input, " "))
	str = strings.Replace(str, "  ", " ", -1)

	if !useColour {
		return str
	}

	// Choose colour for output or none
	switch colour {
	case brightGreen:
		return gchalk.BrightGreen(str)
	case brightYellow:
		return gchalk.BrightYellow(str)
	case brightBlue:
		return gchalk.BrightBlue(str)
	case brightRed:
		return gchalk.BrightRed(str)
	default:
		return str
	}
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

	// var noColourFlag bool
	// flag.BoolVar(&noColourFlag, "C", false, "no colour output")
	// useColour = !noColourFlag

	var stdoutFlag bool
	flag.BoolVar(&stdoutFlag, "S", false, "do not forward standard input to standard output")

	var ignoreFlag bool
	flag.BoolVar(&ignoreFlag, "i", false, "ignore sigint")

	var appendFlag bool
	flag.BoolVar(&appendFlag, "a", false, "append to files if they already exist")

	flag.Parse()
	stdoutFlag = !stdoutFlag

	// args are interpreted as paths
	args := flag.Args()

	if len(args) == 0 {
		out := os.Stderr
		fmt.Fprintln(out, colour(brightRed, "No files specified. Exiting with usage information."))
		printHelp(out)
	}

	// Handle ignoring signals if flag is set
	if ignoreFlag == true {
		ignoreSignal()
	}

	var readWriter *bufio.ReadWriter
	br := bufio.NewReader(os.Stdin)
	bw := bufio.NewWriter(os.Stdout)

	// Use stdin if available
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		readWriter = bufio.NewReadWriter(br, bw)
	} else {
		fmt.Fprintln(os.Stderr, colour(brightRed, "No input. Exiting."))
		printHelp(os.Stderr)
	}

	container := newContainer()
	// Iterate through file path args
	for i := 0; i < len(args); i++ {
		if strings.Contains(args[i], "*") {
			fmt.Fprintln(os.Stderr, "Ignoring globbing path", args[i])
			continue
		}
		_, err := container.addFileWriter(args[i], appendFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Probem obtaining fileWriter for pth", args[i])
		}
	}
	if len(container.writers) == 0 {
		fmt.Fprintln(os.Stderr, "No valid files to save to")
		os.Exit(1)
	}

	buf := make([]byte, 2048)
	count := 0
	for {
		n, err := readWriter.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err.Error())
			break
		}
		if n == 0 {
			break
		}
		// Send bytes to each file fileWriter
		for i := 0; i < len(container.writers); i++ {
			s := container.writers[i]
			if s.active {
				err := s.write(buf[0:n])
				if err != nil {
					s.active = false
				}
			}
		}
		if stdoutFlag {
			readWriter.Write(buf[:n])
		}
		count++
		if err == io.EOF {
			break
		}
	}
	readWriter.Flush()
	for _, s := range container.writers {
		s.close()
	}
}
