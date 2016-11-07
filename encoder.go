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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func waitForInputStream(dir string) (stream os.FileInfo) {
WAIT:
	for {
		files, _ := ioutil.ReadDir(filepath.Clean(dir))
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".ts") {
				stream = f
				break WAIT
			}
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("Found input stream", stream.Name())
	return
}

const (
	videoBitrate = "2250k"
	videoScale   = "-1:680"
	audioBitrate = "128k"
	audioDevice  = "Microphone (Realtek High Defini"
	//audioDevice = "Line-In (Sound Blaster Audigy 5"
)

//Config Metadata Datum
type Config struct {
	Ffmpeg    string   `json:"ffmpeg"`
	FlushSize int      `json:"flushSize"`
	Args      []string `json:"args"`
}

func encoder(conf *Config, dir, output string, stream os.FileInfo) {
	args := append(conf.Args, output)
	cmd := exec.Command(conf.Ffmpeg, args...)

	/*"C:\\ffmpeg\\bin\\ffmpeg.exe", "-itsoffset", "3", "-i", "pipe:",
	"-f", "dshow", "-i", "audio="+audioDevice,
	"-c:v", "libvpx", "-speed", "4", "-threads", "4",
	"-c:a", "libopus", "-b:v", videoBitrate, "-b:a", audioBitrate, "-vf", "scale="+videoScale,
	//"-map", "0:v:0", "-map", "1:a:0" /*, "-r", "30"*/ //, "-f", "webm", output)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if in, err := cmd.StdinPipe(); err != nil {
		log.Fatal(err)
	} else {
		go func() {
			file, err := os.Open(filepath.Join(dir, stream.Name()))
			if err != nil {
				fmt.Println("Could not open", filepath.Join(dir, stream.Name()))
				log.Fatal(err)
			}
			defer file.Close()

			bufferSize := 1024 * 10
			block := make([]byte, bufferSize)
			for {
				read, _ := file.Read(block)
				if read == 0 {
					time.Sleep(time.Millisecond * 10)
				} else if read == bufferSize {
					in.Write(block)
				} else {
					in.Write(block[0:read])
				}
			}
		}()
	}

	fmt.Println("Running FFMPEG with:", cmd.Args)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing ffmpeg", err)
	}
}

func main() {
	flag.Parse()

	dir := "./"
	output := "output.webm"
	if flag.NArg() > 0 {
		output = flag.Arg(0)
	}

	config, err := os.Open("conf.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	var conf Config
	json.NewDecoder(config).Decode(&conf)

	stream := waitForInputStream(dir)
	go encoder(&conf, dir, output, stream)
	serve(output, &conf) //This call blocks by calling HTTP Listen and Serve outside of a go routine
}
