//By TimTheSinner
package main

/**
 * Copyright (c) 2016 TimTheSinner All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
