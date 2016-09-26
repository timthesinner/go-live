package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dir := "./"
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	files, _ := ioutil.ReadDir(filepath.Clean(dir))
	var stream os.FileInfo
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".ts") {
			stream = f
			break
		}
	}

	file, err := os.Open(filepath.Join(dir, stream.Name()))
	if err != nil {
		fmt.Println("Could not open file", err)
	}
	defer file.Close()

	bufferSize := 1024 * 10
	block := make([]byte, bufferSize)
	for {
		read, _ := file.Read(block)
		if read == 0 {
			time.Sleep(time.Millisecond * 10)
		} else if read == bufferSize {
			os.Stdout.Write(block)
		} else {
			os.Stdout.Write(block[0:read])
		}
	}
}
