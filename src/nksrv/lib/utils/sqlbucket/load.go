package sqlbucket

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

type Loader struct {
	name          string
	noNext        bool
	needSemicolon bool
}

func New() Loader {
	return Loader{}
}

func (l Loader) WithName(name string) Loader {
	l.name = name
	return l
}

func (l Loader) WithNoNext(noNext bool) Loader {
	l.noNext = noNext
	return l
}

func (l Loader) WithNeedSemicolon(needSemicolon bool) Loader {
	l.needSemicolon = needSemicolon
	return l
}

func (l Loader) Load(r io.Reader) (queries Bucket, err error) {
	scanner := bufio.NewScanner(r)
	queries = l.Scan(scanner)
	err = scanner.Err()
	return
}

func (l Loader) LoadFromFile(fn string) (_ Bucket, err error) {
	f, err := os.Open(fn)
	if err != nil {
		return
	}
	defer f.Close()

	return l.Load(f)
}

func (l Loader) LoadFromFS(fs fs.FS, name string) (_ Bucket, err error) {
	f, err := fs.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	return l.Load(f)
}

func (l Loader) LoadFromString(s string) (Bucket, error) {
	return l.Load(strings.NewReader(s))
}

func (l Loader) LoadFromBuffer(b []byte) (Bucket, error) {
	return l.Load(bytes.NewBuffer(b))
}

var (
	reName      = regexp.MustCompile(`^\s*--\s*:name\s+(\S+)\s*$`)
	reNameT     = regexp.MustCompile(`^\s*--\s*:namet\s+(\S+)\s*$`)
	reNext      = regexp.MustCompile(`^\s*--\s*:next\s*$`)
	reSet       = regexp.MustCompile(`^\s*--\s*:set\s+(\S+)\s+(\S*)\s*$`)
	reSomething = regexp.MustCompile(`^\s*--\s*:[[:alnum:]]+(?:\s+.*)?\s*$`)
)

func (l Loader) trimFinal(s string) string {
	s = strings.TrimSpace(s)
	if l.needSemicolon {
		if s[len(s)-1] != ';' {
			panic(fmt.Sprintf("no semicolon: %q", s))
		}
		s = s[:len(s)-1]
		s = strings.TrimSpace(s)
	}
	return s
}

func (l Loader) Scan(in *bufio.Scanner) Bucket {
	queries := make(Bucket)

	currtag := l.name
	noName := currtag != ""
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
			queries[currtag][curri] = l.trimFinal(q)
		} else {
			// template
			if curri != 0 {
				panic("can't use :next in conjuction with :namet")
			}
			currt = false
			templates[currtag] = q
			delete(queries, currtag)
		}
	}

	if noName {
		queries[currtag] = []string{""}
	}

	for in.Scan() {

		line := strings.TrimRightFunc(in.Text(), unicode.IsSpace)

		var matches []string

		if !noName {

			matches = reName.FindStringSubmatch(line)
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
		}

		if currtag != "" && !l.noNext && reNext.MatchString(line) {
			finishcurrent()
			// only increase if current non-empty
			if queries[currtag][curri] != "" {
				queries[currtag] = append(queries[currtag], "")
				curri++
			}
			continue
		}

		matches = reSet.FindStringSubmatch(line)
		if len(matches) != 0 {
			key := matches[1]
			queries[key] = append(queries[key], matches[2])
			continue
		}

		if reSomething.MatchString(line) {
			panic(fmt.Sprintf("unrecognised ctl line: %q", line))
		}

		if currtag == "" {
			continue
		}
		queries[currtag][curri] += line + "\n"
	}

	if currtag != "" {
		finishcurrent()
	}

	return queries
}
