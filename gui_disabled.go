//go:build !gui

package main

import "fmt"

const hasGUI = false

func runGUI() {
	fmt.Println("This version of the application was built without GUI support.")
	fmt.Println("To build with the GUI, use the '-tags gui' flag.")
}
