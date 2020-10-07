package main

//concur:main
func foo() int { // want foo:`\[main\]`
	return 0
}

//concur:bar,main
func bar(_ int) { // want bar:`\[bar main\]`
	foo() // want `function foo called from wrong goroutine; bar does not match \[main\]`
}

//concur:bar
func baz() { // want baz:`\[bar\]`
	foo() // want `function foo called from wrong goroutine; bar does not match \[main\]`
	bar(0)

	func() {
		foo() // want `function foo called from wrong goroutine; bar does not match \[main\]`
		bar(0)
	}()

	x := func() int {
		bar(foo())   // want `function foo called from wrong goroutine; bar does not match \[main\]`
		return foo() // want `function foo called from wrong goroutine; bar does not match \[main\]`
	}()

	x = run(func() {
		foo() // want `function foo called from wrong goroutine; bar does not match \[main\]`
	}).(int)
	_ = x

	ch := make(chan func(), 1)
	ch <- quux
	run(<-ch)
}

//concur:quux
func quux() { // want quux:`\[quux\]`
}

//concur:!main
func run(f func()) interface{} { // want run:`\[!main\]`
	f() // want `function quux called from wrong goroutine; bar does not match \[quux\]`
	return 0
}

func init() {
	foo()
	baz() // want `function baz called from wrong goroutine; main does not match \[bar\]`
}

func main() {
	foo()
	bar(0)
	go bar(1)
	baz() // want `function baz called from wrong goroutine; main does not match \[bar\]`
	go baz()
	run(baz) // want `function run called from wrong goroutine; main does not match \[!main\]`
}
