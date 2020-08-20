package sqlbucket

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
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
	reName  = regexp.MustCompile(`^\s*--\s*:name\s*(\S+)\s*$`)
	reNameT = regexp.MustCompile(`^\s*--\s*:namet\s*(\S+)\s*$`)
	reNext  = regexp.MustCompile(`^\s*--\s*:next\s*$`)
)

func Scan(in *bufio.Scanner) Bucket {
	queries := make(Bucket)

	currtag := ""
	curri := 0
	currt := false

	templates := make(map[string]string)

	finishcurrent := func() {
		q := queries[currtag][curri]
		var qw strings.Builder
		e := template.Must(template.New(currtag).Parse(q)).
			Execute(&qw, templates)
		if e != nil {
			panic("exec err: " + e.Error())
		}
		q = qw.String()
		if !currt {
			// normal query
			// XXX improve
			queries[currtag][curri] = strings.TrimSpace(q)
		} else {
			// template
			if curri != 0 {
				panic("cant multitemplate")
			}
			currt = false
			templates[currtag] = q
			delete(queries, currtag)
		}
	}

	for in.Scan() {
		line := strings.TrimRightFunc(in.Text(), unicode.IsSpace)

		matches := reName.FindStringSubmatch(line)
		if len(matches) != 0 {
			if currtag != "" {
				finishcurrent()
			}
			currtag = matches[1]
			queries[currtag] = append(queries[currtag], "")
			curri = len(queries[currtag]) - 1
			continue
		}

		matches = reNameT.FindStringSubmatch(line)
		if len(matches) != 0 {
			if currtag != "" {
				finishcurrent()
			}
			currtag = matches[1]
			currt = true
			queries[currtag] = append(queries[currtag], "")
			curri = len(queries[currtag]) - 1
			continue
		}

		if currtag == "" {
			continue
		}

		if reNext.MatchString(line) {
			finishcurrent()
			// only increase if current non-empty
			if queries[currtag][curri] != "" {
				queries[currtag] = append(queries[currtag], "")
				curri++
			}
			continue
		}

		queries[currtag][curri] += line + "\n"
	}

	if currtag != "" {
		finishcurrent()
	}

	return queries
}
