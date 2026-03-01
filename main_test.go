package main

import "testing"

func TestPrintHello(t *testing.T) {
	got := printHello()
	want := "hello world"

	if got != want {
		t.Errorf("printHello() = %q, want %q", got, want)
	}
}
