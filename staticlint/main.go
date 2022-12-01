package main

import (
	straightExit "github.com/GermanVor/devops-pet-project/staticlint/straightExit"
	"honnef.co/go/tools/staticcheck"

	"github.com/bflad/tfproviderlint/passes/AT001"
	"github.com/bflad/tfproviderlint/passes/AT002"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
)

func main() {
	mychecks := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		straightExit.Analyzer,
		AT001.Analyzer,
		AT002.Analyzer,
	}

	for _, v := range staticcheck.Analyzers {
		mychecks = append(mychecks, v)
	}

	multichecker.Main(mychecks...)
}
