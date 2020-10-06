package main

//tag:foo
func foo() { // want foo:`tags: \[foo\]`
}

//tag:foo
//tag:bar
func bar() { // want bar:`tags: \[foo bar\]`
	foo()
}

func baz() {
	foo() // want "incorrect tags for call of foo: have [], want [foo]"
	bar() // want "incorrect tags for call of bar: have [], want [foo bar]"

	//tag:foo
	func() {
		foo()
	}()
}

//tag:foo
//tag:bar
func main() { // want main:`tags: \[foo bar\]`
	foo()
	bar()
	baz()
}
