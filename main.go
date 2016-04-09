package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Msg struct {
	index int
	str   string
	err   error
}

func lineReader(ch chan<- Msg, index int, rdr io.ReadCloser) {
	defer rdr.Close()

	brdr := bufio.NewReader(rdr)
	for {
		line, err := brdr.ReadString('\n')
		if err != nil {
			ch <- Msg{index: index, err: err}
			return
		}

		ch <- Msg{index: index, str: strings.TrimSpace(line)}
	}
}

func spawnReader(filename string, index int, ch chan<- Msg) error {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: can't open: %s: %v", filename, err)
		return err
	}

	go lineReader(ch, index, f)
	return nil
}

type Format struct {
	pieces	[]string
	inputs	map[int]int	// input index -> piece index
}

func MakeFormat(str string) (Format, error) {
	pieces := []string{}
	inputs := make(map[int]int)

	markerRe, err := regexp.Compile("%([0-9]+)")
	if err != nil {
		return Format{}, err
	}

	for {
		loc := markerRe.FindStringIndex(str)
		if len(loc) == 0 {
			break
		}

		marker := str[loc[0] : loc[1]]

		fileIndex, err := strconv.Atoi(marker[1:])
		if err != nil {
			return Format{}, err
		}

		pieces = append(pieces, str[0 : loc[0]], "")
		inputs[fileIndex] = len(pieces) - 1

		str = str[loc[1] : ]
	}

	if len(str) > 0 {
		pieces = append(pieces, str)
	}

	format := Format {pieces: pieces, inputs: inputs}
	return format, nil
}

func (f Format) String() string {
	return strings.Join(f.pieces, "")
}

func (f *Format) SetInput(index int, content string) {
	f.pieces[f.inputs[index]] = content
}

const usage = "Usage: mux <format string> <file0> [file1 ... fileN]"

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}

	ch := make(chan Msg)

	format, err := MakeFormat(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
		return
	}

	for i := 2; i < len(os.Args); i++ {
		err := spawnReader(os.Args[i], i - 2, ch)
		if err != nil {
			fmt.Fprintln(os.Stderr, "spawn error: ", err)
		}
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}

			if msg.err != nil {
				format.SetInput(msg.index, fmt.Sprintf("(%v)", msg.err))
			} else {
				format.SetInput(msg.index, msg.str)
			}
		}

		fmt.Println(format)
	}
}
