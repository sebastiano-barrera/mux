package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"flag"
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

type Format struct {
	pieces	[]string
	inputs	map[int]int	// input index -> piece index
}

func makeFormat(str string) (Format, error) {
	pieces := []string{}
	inputs := make(map[int]int)

	markerRe, err := regexp.Compile("%([0-9]+)")
	if err != nil {
		return Format{}, err
	}

	for {
		// here, `str` points to the remaining part of the format string;
		loc := markerRe.FindStringIndex(str)
		if len(loc) == 0 {
			break
		}
		
		// `loc[0]` and `loc[1]` are the start and end index of the marker (e.g. "%3") in `str`
		
		marker := str[loc[0] : loc[1]]
		fileIndex, err := strconv.Atoi(marker[1:])
		if err != nil {
			return Format{}, err
		}

		// the empty string will be replaced with a line of text from the corresponding file
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


type IndexSet []int

func (me *IndexSet) String() string {
	var strs []string
	for _, index := range *me {
		strs = append(strs, strconv.Itoa(index))
	}
	return strings.Join(strs, ",")
}

// this method will parse the list of indices passed from the command line
func (me *IndexSet) Set(s string) error {
	iset := IndexSet{}
	toks := strings.Split(s, ",")
	for _, fileIndexStr := range toks {
		fileIndex, err := strconv.Atoi(fileIndexStr)
		if err != nil {
			fmt.Println("parse error: ", err)
			return err
		}
		iset = append(iset, fileIndex)
	}

	*me = iset
	return nil
}

func (me *IndexSet) Find(index int) int {
	for i, item := range *me {
		if index == item {
			return i
		}
	}
	return -1
}


var killingSet = IndexSet{}

func init() {
	flag.Var(&killingSet, "k",
		"Indices of the files which, when closed, cause the program to quit")
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: mux [-k 0,1,...] <format string> <file0> [file1 ... fileN]")
		flag.PrintDefaults()
	}


	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		return
	}

	format, err := makeFormat(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
		return
	}

	// one line-reading coroutine is spawned for each input file;
	// all of them send Msgs to the same channel; 
	// the Msgs are received by the main loop at the end
	
	ch := make(chan Msg)
	var files []io.ReadCloser
	
	args := flag.Args()
	for _, arg := range args[1:] {
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		files = append(files, f)
	}
	
	for i, file := range files {
		go lineReader(ch, i, file)
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}

			if msg.err != nil {
				if msg.err == io.EOF && killingSet.Find(msg.index) != -1 {
					return
				}
				format.SetInput(msg.index, fmt.Sprintf("(%v)", msg.err))
			} else {
				format.SetInput(msg.index, msg.str)
			}
		}

		fmt.Println(format)
	}
}
