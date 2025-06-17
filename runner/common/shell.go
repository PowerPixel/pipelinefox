package common

import (
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

func ParseShell(s string) ([]string, error) {
	p := syntax.NewParser()
	var words []*syntax.Word
	for w, err := range p.WordsSeq(strings.NewReader(s)) {
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	cfg := &expand.Config{Env: nil}
	return expand.Fields(cfg, words...)
}
