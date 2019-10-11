//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// Modified by Martin Atkins to serve the needs of package textseg.

// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var url = flag.String("url",
	"http://www.unicode.org/Public/"+unicode.Version+"/ucd/auxiliary/",
	"URL of Unicode database directory")
var verbose = flag.Bool("verbose",
	false,
	"write data to stdout as it is parsed")
var localFiles = flag.Bool("local",
	false,
	"data files have been copied to the current directory; for debugging only")
var outputFile = flag.String("output",
	"",
	"output file for generated tables; default stdout")

var output *bufio.Writer

func main() {
	flag.Parse()
	setupOutput()

	graphemePropertyRanges := make(map[string]*unicode.RangeTable)
	loadUnicodeData("GraphemeBreakProperty.txt", graphemePropertyRanges)
	wordPropertyRanges := make(map[string]*unicode.RangeTable)
	loadUnicodeData("WordBreakProperty.txt", wordPropertyRanges)
	sentencePropertyRanges := make(map[string]*unicode.RangeTable)
	loadUnicodeData("SentenceBreakProperty.txt", sentencePropertyRanges)

	fmt.Fprintf(output, fileHeader, *url)
	generateTables("Grapheme", graphemePropertyRanges)
	generateTables("Word", wordPropertyRanges)
	generateTables("Sentence", sentencePropertyRanges)

	flushOutput()
}

// WordBreakProperty.txt has the form:
// 05F0..05F2    ; Hebrew_Letter # Lo   [3] HEBREW LIGATURE YIDDISH DOUBLE VAV..HEBREW LIGATURE YIDDISH DOUBLE YOD
// FB1D          ; Hebrew_Letter # Lo       HEBREW LETTER YOD WITH HIRIQ
func openReader(file string) (input io.ReadCloser) {
	if *localFiles {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		input = f
	} else {
		path := *url + file
		resp, err := http.Get(path)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 200 {
			log.Fatal("bad GET status for "+file, resp.Status)
		}
		input = resp.Body
	}
	return
}

func loadUnicodeData(filename string, propertyRanges map[string]*unicode.RangeTable) {
	f := openReader(filename)
	defer f.Close()
	bufioReader := bufio.NewReader(f)
	line, err := bufioReader.ReadString('\n')
	for err == nil {
		parseLine(line, propertyRanges)
		line, err = bufioReader.ReadString('\n')
	}
	// if the err was EOF still need to process last value
	if err == io.EOF {
		parseLine(line, propertyRanges)
	}
}

const comment = "#"
const sep = ";"
const rnge = ".."

func parseLine(line string, propertyRanges map[string]*unicode.RangeTable) {
	if strings.HasPrefix(line, comment) {
		return
	}
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	commentStart := strings.Index(line, comment)
	if commentStart > 0 {
		line = line[0:commentStart]
	}
	pieces := strings.Split(line, sep)
	if len(pieces) != 2 {
		log.Printf("unexpected %d pieces in %s", len(pieces), line)
		return
	}

	propertyName := strings.TrimSpace(pieces[1])

	rangeTable, ok := propertyRanges[propertyName]
	if !ok {
		rangeTable = &unicode.RangeTable{
			LatinOffset: 0,
		}
		propertyRanges[propertyName] = rangeTable
	}

	codepointRange := strings.TrimSpace(pieces[0])
	rngeIndex := strings.Index(codepointRange, rnge)

	if rngeIndex < 0 {
		// single codepoint, not range
		codepointInt, err := strconv.ParseUint(codepointRange, 16, 64)
		if err != nil {
			log.Printf("error parsing int: %v", err)
			return
		}
		if codepointInt < 0x10000 {
			r16 := unicode.Range16{
				Lo:     uint16(codepointInt),
				Hi:     uint16(codepointInt),
				Stride: 1,
			}
			addR16ToTable(rangeTable, r16)
		} else {
			r32 := unicode.Range32{
				Lo:     uint32(codepointInt),
				Hi:     uint32(codepointInt),
				Stride: 1,
			}
			addR32ToTable(rangeTable, r32)
		}
	} else {
		rngeStart := codepointRange[0:rngeIndex]
		rngeEnd := codepointRange[rngeIndex+2:]
		rngeStartInt, err := strconv.ParseUint(rngeStart, 16, 64)
		if err != nil {
			log.Printf("error parsing int: %v", err)
			return
		}
		rngeEndInt, err := strconv.ParseUint(rngeEnd, 16, 64)
		if err != nil {
			log.Printf("error parsing int: %v", err)
			return
		}
		if rngeStartInt < 0x10000 && rngeEndInt < 0x10000 {
			r16 := unicode.Range16{
				Lo:     uint16(rngeStartInt),
				Hi:     uint16(rngeEndInt),
				Stride: 1,
			}
			addR16ToTable(rangeTable, r16)
		} else if rngeStartInt >= 0x10000 && rngeEndInt >= 0x10000 {
			r32 := unicode.Range32{
				Lo:     uint32(rngeStartInt),
				Hi:     uint32(rngeEndInt),
				Stride: 1,
			}
			addR32ToTable(rangeTable, r32)
		} else {
			log.Printf("unexpected range")
		}
	}
}

func addR16ToTable(r *unicode.RangeTable, r16 unicode.Range16) {
	if r.R16 == nil {
		r.R16 = make([]unicode.Range16, 0, 1)
	}
	r.R16 = append(r.R16, r16)
	if r16.Hi <= unicode.MaxLatin1 {
		r.LatinOffset++
	}
}

func addR32ToTable(r *unicode.RangeTable, r32 unicode.Range32) {
	if r.R32 == nil {
		r.R32 = make([]unicode.Range32, 0, 1)
	}
	r.R32 = append(r.R32, r32)
}

func generateTables(prefix string, propertyRanges map[string]*unicode.RangeTable) {
	prNames := make([]string, 0, len(propertyRanges))
	for k := range propertyRanges {
		prNames = append(prNames, k)
	}
	sort.Strings(prNames)
	for _, key := range prNames {
		rt := propertyRanges[key]
		fmt.Fprintf(output, "var _%s%s = %s\n", prefix, key, generateRangeTable(rt))
	}
	fmt.Fprintf(output, "type _%sRuneRange unicode.RangeTable\n", prefix)

	fmt.Fprintf(output, "func _%sRuneType(r rune) *_%sRuneRange {\n", prefix, prefix)
	fmt.Fprintf(output, "\tswitch {\n")
	for _, key := range prNames {
		fmt.Fprintf(output, "\tcase unicode.Is(_%s%s, r):\n\t\treturn (*_%sRuneRange)(_%s%s)\n", prefix, key, prefix, prefix, key)
	}
	fmt.Fprintf(output, "\tdefault:\n\t\treturn nil\n")
	fmt.Fprintf(output, "\t}\n")
	fmt.Fprintf(output, "}\n")

	fmt.Fprintf(output, "func (rng *_%sRuneRange) String() string {\n", prefix)
	fmt.Fprintf(output, "\tswitch (*unicode.RangeTable)(rng) {\n")
	for _, key := range prNames {
		fmt.Fprintf(output, "\tcase _%s%s:\n\t\treturn %q\n", prefix, key, key)
	}
	fmt.Fprintf(output, "\tdefault:\n\t\treturn \"Other\"\n")
	fmt.Fprintf(output, "\t}\n")
	fmt.Fprintf(output, "}\n")
}

func generateRangeTable(rt *unicode.RangeTable) string {
	rv := "&unicode.RangeTable{\n"
	if rt.R16 != nil {
		rv += "\tR16: []unicode.Range16{\n"
		for _, r16 := range rt.R16 {
			rv += fmt.Sprintf("\t\t%#v,\n", r16)
		}
		rv += "\t},\n"
	}
	if rt.R32 != nil {
		rv += "\tR32: []unicode.Range32{\n"
		for _, r32 := range rt.R32 {
			rv += fmt.Sprintf("\t\t%#v,\n", r32)
		}
		rv += "\t},\n"
	}
	rv += fmt.Sprintf("\t\tLatinOffset: %d,\n", rt.LatinOffset)
	rv += "}\n"
	return rv
}

const fileHeader = `// Generated by running
//      maketables --url=%s
// DO NOT EDIT

package textseg

import(
	"unicode"
)
`

func setupOutput() {
	output = bufio.NewWriter(startGofmt())
}

// startGofmt connects output to a gofmt process if -output is set.
func startGofmt() io.Writer {
	if *outputFile == "" {
		return os.Stdout
	}
	stdout, err := os.Create(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	// Pipe output to gofmt.
	gofmt := exec.Command("gofmt")
	fd, err := gofmt.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	gofmt.Stdout = stdout
	gofmt.Stderr = os.Stderr
	err = gofmt.Start()
	if err != nil {
		log.Fatal(err)
	}
	return fd
}

func flushOutput() {
	err := output.Flush()
	if err != nil {
		log.Fatal(err)
	}
}
