// TODO: allow tagging functions or entire packages through a config file

package concur

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

var Analyzer = &analysis.Analyzer{
	Name: "concur",
	Doc:  "check that certain functions are only called from the correct goroutine",
	Run:  run,

	Requires:  []*analysis.Analyzer{buildssa.Analyzer},
	FactTypes: []analysis.Fact{new(tagFact)},
}

func run(pass *analysis.Pass) (interface{}, error) {
	c := concurChecker{pass, nil}

	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			if decl, ok := decl.(*ast.FuncDecl); ok {
				c.emitTags(decl)
			}
		}
	}

	ssaRes := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if len(ssaRes.SrcFuncs) == 0 {
		return nil, nil
	}
	g := rta.Analyze(ssaRes.SrcFuncs, true).CallGraph

	for _, node := range g.Nodes {
		name := node.Func.Name()
		if name == "main" || strings.HasPrefix(name, "init#") {
			tags := c.getTags(node.Func)
			edge := &callgraph.Edge{Callee: node}
			if len(tags) > 0 {
				c.visit(edge, nil, nil, tags[0])
			} else {
				c.visit(edge, nil, nil, "main")
			}
		} else {
			for _, edge := range node.In {
				if _, ok := edge.Site.(*ssa.Go); ok {
					c.visit(edge, nil, nil, c.concurName(node.Func))
					break
				}
			}
		}
	}

	for _, diag := range c.msgs {
		pass.Report(diag)
	}
 
	return nil, nil
}

type tagFact []string

func (f tagFact) String() string {
	return fmt.Sprint([]string(f))
}
func (tagFact) AFact() {}

func commentTags(group *ast.CommentGroup) (tags []string) {
	for _, comment := range group.List {
		if strings.HasPrefix(comment.Text, "//concur:") {
			tag := strings.TrimPrefix(comment.Text, "//concur:")
			if tag != "" {
				tags = append(tags, strings.Split(tag, ",")...)
			}
		}
	}
	return tags
}

type concurChecker struct {
	pass *analysis.Pass
	msgs []analysis.Diagnostic
}

func (c concurChecker) emitTags(decl *ast.FuncDecl) {
	if decl.Doc == nil {
		return
	}

	tags := commentTags(decl.Doc)
	if len(tags) > 0 {
		fact := tagFact(tags)
		c.pass.ExportObjectFact(c.pass.TypesInfo.Defs[decl.Name], &fact)
	}
}

func (c concurChecker) getTags(fun *ssa.Function) []string {
	obj := fun.Object()
	if obj == nil {
		return nil
	}
	var fact tagFact
	c.pass.ImportObjectFact(obj, &fact)
	return []string(fact)
}

func (c concurChecker) concurName(fun *ssa.Function) string {
	tags := c.getTags(fun)
	if len(tags) > 0 && tags[0][0] != '!' {
		return tags[0]
	} else {
		return fun.Name()
	}
}

func inRuntime(fun *ssa.Function) bool {
	return fun.Pkg != nil && fun.Pkg.Pkg != nil && fun.Pkg.Pkg.Path() == "runtime"
}

func inPath(fun *ssa.Function, path []*ssa.Function) bool {
	for _, pfun := range path {
		if fun == pfun {
			return true
		}
	}
	return false
}

func (c *concurChecker) visit(edge *callgraph.Edge, path []*ssa.Function, gopath []*ssa.Function, name string) {
	// Special case to skip runtime since it's pointless to spend ages checking it
	if inRuntime(edge.Callee.Func) {
		return
	}

	if !c.check(edge, name) {
		return
	}

	if inPath(edge.Callee.Func, path) {
		return
	}

	path = append(path, edge.Callee.Func)
	for _, child := range edge.Callee.Out {
		if _, ok := child.Site.(*ssa.Go); ok {
			if !inPath(child.Callee.Func, gopath) {
				newName := c.concurName(child.Callee.Func)
				newGopath := append(gopath, edge.Callee.Func)
				c.visit(child, nil, newGopath, newName)
			}
		} else {
			c.visit(child, path, gopath, name)
		}
	}
}

func (c *concurChecker) check(edge *callgraph.Edge, name string) bool {
	tags := c.getTags(edge.Callee.Func)

	check := false
	ok := false
	for _, tag := range tags {
		if tag[0] == '!' {
			if tag == "!"+name {
				// Restriction broken
				check = true
				ok = false
				break
			}
		} else {
			check = true
			if tag == name {
				// Restriction met
				ok = true
			}
		}
	}

	if !check || ok {
		// All restrictions okay
		return true
	}

	msg := fmt.Sprintf(
		"function %s called from wrong goroutine; %s does not match %v",
		edge.Callee.Func.Name(), name, tags,
	)
	c.report(analysis.Diagnostic{
		Pos:     edge.Pos(),
		Message: msg,
	})

	return false
}

func (c *concurChecker) report(diag analysis.Diagnostic) {
	// Search
	var i int
	for i = 0; i < len(c.msgs) && c.msgs[i].Pos <= diag.Pos; i++ {
		if c.msgs[i].Pos == diag.Pos && c.msgs[i].Message == diag.Message {
			// Already exists, don't duplicate
			return
		}
	}

	// Insert
	c.msgs = append(c.msgs, analysis.Diagnostic{})
	copy(c.msgs[i+1:], c.msgs[i:])
	c.msgs[i] = diag
}
