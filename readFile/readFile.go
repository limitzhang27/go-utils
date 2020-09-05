package readFile

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
)

func read1(filename string) int {
	fp, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	buf := make([]byte, 4096)
	var nbytes int
	for {
		n, err := fp.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		nbytes += n
	}
	return nbytes
}

func read2(filename string) int {
	fp, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	rd := bufio.NewReader(fp)
	var nbytes int
	buf := make([]byte, 4096)
	for {
		n, err := rd.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}

		if n == 0 {
			break
		}
		nbytes += n
	}
	return nbytes
}

func read3(filename string) int {
	fp, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	data, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(err)
	}
	return len(data)
}
