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

// fileWriter struct to help manage writing to a file
type fileWriter struct {
	file   *os.File
	writer *bufio.Writer
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
func (s *fileWriter) write(bytes []byte) {
	if _, err := s.writer.Write(bytes); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// close close the underlying writer
func (s *fileWriter) close() {
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// container holds slice of fileWriters
type container struct {
	savers []*fileWriter
}

// newContainer properly initialize a new container
func newContainer() *container {
	c := new(container)
	c.savers = make([]*fileWriter, 0, 5)

	return c
}

// addFileWriter add a saver to the container's slice
func (c *container) addFileWriter(s *fileWriter) {
	c.savers = append(c.savers, s)
}

// write write incoming bytes to all savers
func (c *container) write(bytes []byte) {
	for _, s := range c.savers {
		s.write(bytes)
	}
}

// close call close on all savers
func (c *container) close() {
	for _, s := range c.savers {
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
	fmt.Fprintln(out, colour(brightGreen, os.Args[0], "- a simple tail program"))
	fmt.Fprintln(out, "Usage")
	fmt.Fprintln(out, "- print tail (or head) n lines of one or more files")
	fmt.Fprintln(out, "Example: tail -n 10 file1.txt file2.txt")
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
	var ignoreFlag bool
	flag.BoolVar(&ignoreFlag, "i", false, "ignore sigint")

	var appendFlag bool
	flag.BoolVar(&appendFlag, "a", false, "append")

	flag.Parse()

	c := make(chan os.Signal, 1)
	if ignoreFlag == true {
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				fmt.Fprintln(os.Stderr, colour(brightRed, "Got", sig.String()))
			}
		}()
	}

	// args are interpreted as paths
	args := flag.Args()

	if len(args) == 0 {
		out := os.Stderr
		fmt.Fprintln(out, colour(brightRed, "No files specified. Exiting with usage information."))
		printHelp(out)
	}

	var readWriter *bufio.ReadWriter
	br := bufio.NewReader(os.Stdin)
	bw := bufio.NewWriter(os.Stdout)
	// Use stdin if available
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		readWriter = bufio.NewReadWriter(br, bw)
	} else {
		fmt.Fprintln(os.Stderr, "No input")
		printHelp(os.Stderr)
	}

	container := newContainer()
	// Iterate through file path args
	for i := 0; i < len(args); i++ {
		if strings.Contains(args[i], "*") {
			continue
		}
		saver, err := newFileWriter(args[i], appendFlag)
		// fmt.Println("Adding for file", args[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Probem obtaining saver for pth", args[i])
			continue
		}
		container.addFileWriter(saver)
	}
	if len(container.savers) == 0 {
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
		// Send bytes to each file saver
		for i := 0; i < len(container.savers); i++ {
			s := container.savers[i]
			s.write(buf[0:n])
		}
		readWriter.Write(buf[:n])
		count++
		if err == io.EOF {
			break
		}
	}
	readWriter.Flush()
	for _, s := range container.savers {
		s.close()
	}
}
