package main

import (
	straightExit "github.com/GermanVor/devops-pet-project/staticlint/straightExit"
	"github.com/bflad/tfproviderlint/passes/AT001"
	"github.com/bflad/tfproviderlint/passes/AT002"
	"honnef.co/go/tools/staticcheck"

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
		staticcheck.Analyzers["SA1001"],
		staticcheck.Analyzers["SA1013"],
		staticcheck.Analyzers["SA1016"],
		staticcheck.Analyzers["SA1019"],
		staticcheck.Analyzers["SA2000"],
		staticcheck.Analyzers["SA2001"],
		staticcheck.Analyzers["SA3000"],
		staticcheck.Analyzers["SA3001"],
		staticcheck.Analyzers["SA4000"],
		staticcheck.Analyzers["SA4001"],
		staticcheck.Analyzers["SA4003"],
		// staticcheck.Analyzers["SA6003"],
	}

	// for _, v := range staticcheck.Analyzers {
	// 	mychecks = append(mychecks, v)
	// }

	multichecker.Main(mychecks...)
}
