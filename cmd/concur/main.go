package main

import (
	"github.com/vktec/concur"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(concur.Analyzer) }
