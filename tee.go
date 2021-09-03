package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
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

type saver struct {
	file *os.File

	// input  chan []byte
	// done   chan struct{}
	writer *bufio.Writer
}

func newSaver(path string, append bool) (*saver, error) {
	s := new(saver)

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
	}

	s.file, err = os.OpenFile(path, mode|os.O_WRONLY, 0644)
	if err != nil {
		// Something wrong like bad file path
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, err
	}
	s.writer = bufio.NewWriter(s.file)

	// go func() {
	// 	// close fi on exit and check for its returned error
	// 	defer func() {
	// 		if err := s.file.Close(); err != nil {
	// 			panic(err)
	// 		}
	// 	}()
	// 	// make a write buffer
	// 	// buf := make([]byte, 1024)

	// 	for bytes := range s.input {
	// 		// write a chunk
	// 		if _, err := s.writer.Write(bytes); err != nil {
	// 			panic(err)
	// 		}
	// 		// if err := s.writer.Flush(); err != nil {
	// 		// 	panic(err)
	// 		// }
	// 	}
	// }()

	return s, nil
}

func (s *saver) write(bytes []byte) {

	// n, _ := s.file.Seek(0, os.SEEK_END)

	// fmt.Println(n)

	// _, err := s.file.WriteAt(bytes, n)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// }

	s.file.Write(bytes)
	// // write a chunk
	// fmt.Fprint(s.file, string(bytes))
	// if _, err := s.writer.Write(bytes); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// }
	// if err := s.writer.Flush(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// }
}

func (s *saver) close() {
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

type container struct {
	savers []*saver
	done   chan struct{}
}

func newContainer() *container {
	c := new(container)
	c.savers = make([]*saver, 0, 5)
	c.done = make(chan struct{})

	return c
}

func (c *container) addSaver(s *saver) {
	c.savers = append(c.savers, s)
}

func (c *container) write(bytes []byte) {
	for _, s := range c.savers {
		s.write(bytes)
	}
}

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

	var noColourFlag bool
	flag.BoolVar(&noColourFlag, "C", false, "no colour output")

	useColour = !noColourFlag

	var appendFlag bool
	flag.BoolVar(&appendFlag, "a", false, "append")

	flag.Parse()

	// args are interpreted as paths
	args := flag.Args()

	if len(args) == 0 {
		out := os.Stderr
		fmt.Fprintln(out, colour(brightRed, "No files specified. Exiting with usage information."))
		printHelp(out)
	}

	var reader *bufio.Reader
	// Use stdin if available
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader = bufio.NewReader(os.Stdin)
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
		saver, err := newSaver(args[i], appendFlag)
		// fmt.Println("Adding for file", args[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Probem obtaining saver for pth", args[i])
			continue
		}
		container.addSaver(saver)
	}
	if len(container.savers) == 0 {
		fmt.Fprintln(os.Stderr, "No valid files to save to")
		os.Exit(1)
	}

	buf := make([]byte, 1024)
	count := 0
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		// Send bytes to each file saver
		for i := 0; i < len(container.savers); i++ {
			s := container.savers[i]
			s.write(buf[0:n])
		}
		count++
	}
	for _, s := range container.savers {
		s.close()
	}
}
