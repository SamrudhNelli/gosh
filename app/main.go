package main

import (
	"fmt"
	"bufio"
	"os"
	"unicode"
)

func main() {   
	for {
		fmt.Print("$ ")
		command, error := bufio.NewReader(os.Stdin).ReadString('\n')
		if error != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", error)
			os.Exit(1)
		}
		if command == "exit\n" { 
			break 
		} else if command[:4] == "echo" && unicode.IsSpace(rune(command[4])) {
			if len(command) == 5 {
				fmt.Println()
			} else {
				fmt.Print(command[5:len(command)])
			}
		} else {
			fmt.Println(command[0:(len(command) - 1)] + ": command not found")
		}
	}
}