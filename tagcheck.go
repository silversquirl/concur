package tagcheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "tagcheck",
	Doc:  "check that tagged functions are only called from functions with the same tags",
	Run:  run,

	FactTypes: []analysis.Fact{new(tagFact)},
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			if decl, ok := decl.(*ast.FuncDecl); ok {
				emitTags(pass, decl)
			}
		}
	}

	for _, f := range pass.Files {
		ast.Walk(tagCheckVisitor{pass, f, nil}, f)
	}

	return nil, nil
}

type tagFact []string

func (f tagFact) String() string {
	return fmt.Sprintf("tags: %v", []string(f))
}
func (tagFact) AFact() {}

func commentTags(group *ast.CommentGroup) (tags []string) {
	for _, comment := range group.List {
		if strings.HasPrefix(comment.Text, "//tag:") {
			tag := strings.TrimPrefix(comment.Text, "//tag:")
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func emitTags(pass *analysis.Pass, decl *ast.FuncDecl) {
	if decl.Doc == nil {
		return
	}

	tags := commentTags(decl.Doc)
	if len(tags) > 0 {
		fact := tagFact(tags)
		pass.ExportObjectFact(pass.TypesInfo.Defs[decl.Name], &fact)
	}
}

type tagCheckVisitor struct {
	pass *analysis.Pass
	file *ast.File
	tags []string
}

func (v tagCheckVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.CallExpr:
		if ident, ok := node.Fun.(*ast.Ident); ok {
			tags := v.objTags(v.pass.TypesInfo.Uses[ident])
			if !checkTags(v.tags, tags) {
				msg := fmt.Sprintf("incorrect tags for call of %s: have %v, want %v", ident.Name, v.tags, tags)
				v.pass.Report(analysis.Diagnostic{
					Pos:     node.Lparen,
					Message: msg,
					// TODO: suggest a fix
				})
			}
		}

	case *ast.FuncDecl:
		return tagCheckVisitor{v.pass, v.file, v.objTags(v.pass.TypesInfo.Defs[node.Name])}

	case *ast.FuncLit:
		return tagCheckVisitor{v.pass, v.file, v.lineTags(node.Pos())}
	}
	return v
}

func (v tagCheckVisitor) objTags(obj types.Object) []string {
	var fact tagFact
	v.pass.ImportObjectFact(obj, &fact)
	return []string(fact)
}

func (v tagCheckVisitor) lineTags(pos token.Pos) (tags []string) {
	file := v.pass.Fset.File(pos)
	line := file.Line(pos)
	for _, group := range v.file.Comments {
		cline := file.Line(group.End())
		if line == cline || line-1 == cline {
			tags = append(tags, commentTags(group)...)
		}
	}
	return tags
}

// checkTags checks that all tags in want also exist in have.
// This function takes O(nm) time, but n and m are normally small so it's fine.
func checkTags(have, want []string) bool {
	if len(have) < len(want) {
		return false
	}
	for _, wanted := range want {
		if !checkTag(wanted, have) {
			return false
		}
	}
	return true
}

func checkTag(wanted string, tags []string) bool {
	for _, tag := range tags {
		if wanted == tag {
			return true
		}
	}
	return false
}
