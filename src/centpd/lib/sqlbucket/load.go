package sqlbucket

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

func Load(r io.Reader) (queries Bucket, err error) {
	scanner := bufio.NewScanner(r)
	queries = Scan(scanner)
	err = scanner.Err()
	return
}

func LoadFromFile(fn string) (_ Bucket, err error) {
	f, err := os.Open(fn)
	if err != nil {
		return
	}
	defer f.Close()

	return Load(f)
}

func LoadFromString(s string) (Bucket, error) {
	return Load(strings.NewReader(s))
}

func LoadFromBuffer(b []byte) (Bucket, error) {
	return Load(bytes.NewBuffer(b))
}

var (
	reName = regexp.MustCompile(`^\s*--\s*:name\s*(\S+)\s*$`)
	reNext = regexp.MustCompile(`^\s*--\s*:next\s*$`)
)

func Scan(in *bufio.Scanner) Bucket {
	queries := make(Bucket)

	currtag := ""
	curri := 0

	cleancurrent := func() {
		queries[currtag][curri] = strings.TrimSpace(queries[currtag][curri])
	}

	for in.Scan() {
		line := strings.TrimRightFunc(in.Text(), unicode.IsSpace)

		matches := reName.FindStringSubmatch(line)
		if len(matches) != 0 {
			if currtag != "" {
				cleancurrent()
			}
			currtag = matches[1]
			queries[currtag] = append(queries[currtag], "")
			curri = len(queries[currtag]) - 1
			continue
		}

		if currtag == "" {
			continue
		}

		if reNext.MatchString(line) {
			cleancurrent()
			queries[currtag] = append(queries[currtag], "")
			curri++
			continue
		}

		queries[currtag][curri] += line + "\n"
	}

	if currtag != "" {
		cleancurrent()
	}

	return queries
}
