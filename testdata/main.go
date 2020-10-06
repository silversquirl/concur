package main

//tag:foo
func foo() int { // want foo:`tags: \[foo\]`
	return 0
}

//tag:foo
//tag:bar
func bar(_ int) { // want bar:`tags: \[foo bar\]`
	foo()
}

func baz() {
	foo()  // want "incorrect tags for call of foo: have [], want [foo]"
	bar(0) // want "incorrect tags for call of bar: have [], want [foo bar]"

	func() {
		foo()  // want "incorrect tags for call of foo: have [], want [foo]"
		bar(0) // want "incorrect tags for call of bar: have [], want [foo bar]"
	}()

	//tag:foo
	//tag:bar
	x := func() int {
		bar(foo())
		return foo()
	}()

	go func() { //tag:foo
		foo()
	}()

	x = run(func() { //tag:foo
		foo()
	}).(int)
	_ = x
}

func run(f func()) interface{} {
	f()
	return 0
}

//tag:foo
//tag:bar
func main() { // want main:`tags: \[foo bar\]`
	foo()
	bar(0)
	baz()
}
