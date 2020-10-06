package main

import (
	"github.com/vktec/tagcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(tagcheck.Analyzer) }
