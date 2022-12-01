package straightexit

import (
	"go/ast"
	"log"

	"golang.org/x/tools/go/analysis"
)

const Doc = `The straightexit forbides using a direct call of os.Exit in the main function of the main package.`
const Message = `it is forbidden to use a direct call to os.Exit in the main function of the main package`
const Category = `Illegal using os.Exit`

func getSelectorExpr(stmt ast.Stmt) *ast.SelectorExpr {
	if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
				return fun
			}
		}
	}

	return nil
}

var Analyzer = &analysis.Analyzer{
	Name: "straightexit",
	Doc:  Doc,
	Run: func(p *analysis.Pass) (interface{}, error) {
		log.Println("straightexit:", "Dir: ", p)

		for _, file := range p.Files {
			log.Println("\t", "package:", file.Name.Name)

			// check if package main
			if file.Name.Name != "main" {
				continue
			}

			// check if package imports "os"
			isOsImported := false
			for _, a := range file.Imports {
				if a.Path.Value == `"os"` {
					isOsImported = true
					break
				}
			}
			if !isOsImported {
				log.Println("\t", "package:", file.Name.Name, "no os package")
				continue
			}

			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if x.Name.Name != "main" && x.Recv != nil {
						return true
					}

					for _, stmt := range x.Body.List {
						fun := getSelectorExpr(stmt)
						if fun == nil {
							continue
						}

						if fun.Sel.Name == "Exit" {
							p.Report(analysis.Diagnostic{
								Pos:      fun.Pos(),
								End:      fun.End(),
								Message:  Message,
								Category: Category,
							})
						}
					}
				}
				return true
			})
		}

		return nil, nil
	},
}
