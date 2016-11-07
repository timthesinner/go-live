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
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxBlockSize = 1024 * 1024       //Max block size set to 1MB
const startBlockSize = 1024 * 1024 * 2 //Starting block size set to 2MB
const chunkRequest = 1024 * 1024 * 3   //Offset from "now" for folks that join late 3MB

const defaultFlushSize = 1024 * 45                     // Flush 40KB at a time
const flushTime time.Duration = 100 * time.Millisecond //Tick every 100ms

var _range = regexp.MustCompile(`bytes=(\d+)-(\d*)`)
var _head = regexp.MustCompile(`^.*?-(\d+)$`)

type webmStream struct {
	stream    string
	flushSize int
}

func newWebmStream(stream string, flushSize int) *webmStream {
	return &webmStream{stream, flushSize}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func rangeRequest(fileSize int64, req *http.Request) (int64, int64) {
	offset := int64(0)
	blockSize := int64(maxBlockSize)
	if r := _range.FindStringSubmatch(req.Header.Get("Range")); len(r) != 0 {
		var err error
		if offset, err = strconv.ParseInt(r[1], 10, 64); err != nil {
			offset = 0
		}

		if strings.HasPrefix(req.Header.Get("User-Agent"), "Lavf") {
			blockSize = fileSize - offset
		} else if end, err := strconv.ParseInt(r[2], 10, 64); err == nil {
			blockSize = end - offset + 1
		} else if offset == 0 {
			blockSize = startBlockSize
		} else {
			blockSize = min(maxBlockSize, fileSize-offset)
		}

		if blockSize < 0 {
			blockSize = int64(maxBlockSize)
		}
	}
	return offset, blockSize
}

type flusher struct {
	f         http.Flusher
	w         http.ResponseWriter
	start     bool
	flushSize int
}

func newFlusher(w http.ResponseWriter, blockSize int64, flushSize int) (flusher, error) {
	if f, ok := w.(http.Flusher); ok {
		return flusher{w: w, f: f, start: blockSize == startBlockSize, flushSize: flushSize}, nil
	}

	return flusher{}, errors.New("Response Writer was not a flusher")
}

func (f flusher) Write(b []byte) (n int, err error) {
	length := len(b)
	flushSize := f.flushSize
	blocks := length / flushSize
	var _n int

	ticks := flushTime
	if f.start { //Double the transfer rate to fast-fill the client buffer on startup
		ticks = ticks / 2
	}

	i := 0
	for range time.Tick(ticks) {
		if i < blocks {
			_n, err = f.w.Write(b[n : n+flushSize])
			n += _n

			if err != nil || _n != flushSize {
				fmt.Println("Err", err, "N", _n, "Flush", flushSize)
				return
			}

			f.f.Flush()
			i++
		} else {
			break
		}
	}

	if n != length {
		extra := length % flushSize
		_n, err = f.w.Write(b[n:length])
		n += _n

		if err != nil || _n != extra {
			return
		}

		f.f.Flush()
	}

	return n, nil
}

func (s *webmStream) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

	file, err := os.Open(filepath.Clean(s.stream))
	if err != nil {
		fmt.Println("Could not open file", err)
	}
	defer file.Close()

	stat, _ := file.Stat()
	fileSize := stat.Size()
	offset, blockSize := rangeRequest(fileSize, req)
	headerOffset := offset

	//Stream head offset so folks that join the stream late start "now"
	if r := _head.FindStringSubmatch(req.URL.Path); len(r) != 0 {
		if head, err2 := strconv.ParseInt(r[1], 10, 64); err2 == nil {
			if offset == 0 {
				blockSize = _StreamHead.InitializationSegment
			} else {
				file.Seek(head, 0)
				offset -= _StreamHead.InitializationSegment
				if offset == 0 {
					blockSize = startBlockSize
				}
			}
		}
	}

	if offset != 0 {
		file.Seek(offset, 1)
	}

	block := make([]byte, blockSize)
	read, _ := file.Read(block)
	if read == 0 {
		for read == 0 {
			time.Sleep(time.Millisecond * 100)
			read, _ = file.Read(block)
		}
	}

	read64 := int64(read)

	res.Header().Add("Accept-Ranges", "bytes")
	res.Header().Add("Content-Type", "video/mp4")
	res.Header().Add("Content-Length", strconv.Itoa(read))
	//Setting file size to a hard coded 3780MB which is approx 3 hours in the future
	res.Header().Add("Content-Range", "bytes "+strconv.FormatInt(headerOffset, 10)+"-"+strconv.FormatInt(headerOffset+read64-1, 10)+"/30240000000")
	res.WriteHeader(http.StatusPartialContent)

	var writer io.Writer
	if writer, err = newFlusher(res, blockSize, s.flushSize); err != nil {
		writer = res
	}

	if read64 == blockSize {
		writer.Write(block)
	} else {
		//Hold 1 second to build up more data in the file buffer
		time.Sleep(time.Second * 1)
		writer.Write(block[0:read])
	}
}

type streamHead struct {
	stream string
}

func newStreamHead(stream string) *streamHead {
	return &streamHead{stream}
}

type videoDatum struct {
	Name                  string  `json:"name"`
	Head                  int64   `json:"head"`
	Buffer                []int64 `json:"buffer"`
	InitializationSegment int64   `json:"init"`
}

var _StreamHead = videoDatum{}

func (s *streamHead) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	res.Header().Add("Content-Type", "Application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(_StreamHead)
}

func monitor(stream string) {
	file, err := os.Open(filepath.Clean(stream))
	if err != nil {
		fmt.Println("Could not open file", err)
	}
	defer file.Close()

	stat, _ := file.Stat()
	_StreamHead.Name = stat.Name()
	streamBufferSize := 6
	_StreamHead.Buffer = make([]int64, streamBufferSize)

	block := make([]byte, 1024*64)
	window := make([]byte, 4)
	index := int64(-4)
	for {
		read, _ := file.Read(block)
		if read == 0 {
			time.Sleep(time.Millisecond * 250)
		} else {
			for i := 0; i < read; i++ {
				index++
				window[0] = window[1]
				window[1] = window[2]
				window[2] = window[3]
				window[3] = block[i]
				//Use a rolling window to find the file offset of the latest Cluster header
				// Folks that join the stream will join from the last cluster header, this is as close to "now" as they can get
				if window[0] == 0x1F && window[1] == 0x43 && window[2] == 0xB6 && window[3] == 0x75 {
					if _StreamHead.InitializationSegment == 0 {
						_StreamHead.InitializationSegment = index
						for i := 0; i < streamBufferSize; i++ {
							_StreamHead.Buffer[i] = index
						}
					}
					for i, j := 0, 1; j < streamBufferSize; i, j = i+1, j+1 {
						_StreamHead.Buffer[i] = _StreamHead.Buffer[j]
					}
					_StreamHead.Buffer[streamBufferSize-1] = index
					_StreamHead.Head = _StreamHead.Buffer[0]
				}
			}
		}
	}
}

func waitForOutputStream(output string) {
	for {
		if _, err := os.Stat(output); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	fmt.Println("Found output stream", output)
}

func serve(output string, conf *Config) {
	waitForOutputStream(output)

	if conf.FlushSize < defaultFlushSize {
		conf.FlushSize = defaultFlushSize
	}

	go monitor(output)

	http.Handle("/", newTemplateServer())
	http.Handle("/webm/", http.StripPrefix("/webm/", newWebmStream(output, conf.FlushSize)))
	http.Handle("/rest/stream-head", http.StripPrefix("/rest/stream-head", newStreamHead(output)))
	log.Fatal(http.ListenAndServeTLS(":8080", "server.pem", "server.key", nil))
}

type templateServer struct{}

func newTemplateServer() *templateServer {
	return &templateServer{}
}

var _ClientMap = make(map[string]string)

func (s *templateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

	const html = `<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>Stream It</title>
	</head>
	<body>
		<video controls width="80%" autostart autoplay>
			<source src="/webm/{{.Name}}-{{.Head}}" type="video/webm">
			<p>Your browser does not support embedded videos.  Metadata: {{.Buffer}}</p>
		</video>
	</body>
</html>
`

	if t, err := template.New("server").Parse(html); err != nil {
		fmt.Println("error parsing", err)
	} else if err = t.Execute(w, _StreamHead); err != nil {
		fmt.Println("Error executing", err)
	}

	if _, ok := _ClientMap[r.RemoteAddr]; !ok {
		fmt.Println("New Client", r.RemoteAddr)
		_ClientMap[r.RemoteAddr] = time.Now().String()
	}
}
