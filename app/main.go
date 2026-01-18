package main

import (
	"fmt"
	"bufio"
	"os"
)

func main() {   
	for {
		fmt.Print("$ ")
		command, error := bufio.NewReader(os.Stdin).ReadString('\n')
		var _ = error
		fmt.Println(command[0:(len(command) - 1)] + ": command not found")
	}
}