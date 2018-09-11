package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/arjantop/pwned-passwords/internal/storage"
)

var outputDir = flag.String("outputDir", "", "Output directory for pre-processed files")

type prefixWriter struct {
	PrefixLength   int
	PathPartLength int
	currentPrefix  string
	currentFile    *os.File
}

func (w *prefixWriter) WriteHash(hash string) error {
	prefix := hash[0:5]
	if w.currentPrefix != prefix {
		w.currentPrefix = prefix

		filePath := storage.PathFor(prefix, ".bin")

		fullPath := path.Join(*outputDir, filePath)
		err := os.MkdirAll(path.Dir(fullPath), 0755)
		if err != nil {
			return fmt.Errorf("creating directory failed: %s", err)
		}

		f, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("creating file failed: %s", err)
		}
		w.currentFile = f
	}

	h, err := hex.DecodeString(hash)
	if err != nil {
		return fmt.Errorf("decoding hash failed: %s", err)
	}

	if _, err := w.currentFile.Write(h); err != nil {
		return fmt.Errorf("writing hash failed: %s", err)
	}

	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 || *outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	fileName := flag.Arg(0)

	var input io.ReadCloser
	if fileName == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(fileName)
		if err != nil {
			log.Fatalf("Could not open file: %s", err)
		}
		input = f
	}
	defer input.Close()

	prefixWriter := &prefixWriter{
		PrefixLength:   5,
		PathPartLength: 3,
	}
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			log.Fatalf("Unexpected line: %s", line)
		}

		if err := prefixWriter.WriteHash(parts[0]); err != nil {
			log.Fatalf("Could not write line: %s", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Could not read from file: %s", err)
	}
}
