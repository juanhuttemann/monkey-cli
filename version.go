package main

import _ "embed"

// AppName is the application name.
const AppName = "monkey"

// AppTitle is the display name used in the intro block title.
const AppTitle = "monkey CLI"

// Version is the application version.
const Version = "0.2.0"

//go:embed intro.txt
var introArt string

// introContent returns the ASCII art shown inside the intro block.
func introContent() string {
	return introArt
}
