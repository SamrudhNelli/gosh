package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	for {
		fmt.Print("$ ")
		rawCommand, error := bufio.NewReader(os.Stdin).ReadString('\n')
		command := strings.Fields(rawCommand)

		if error != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", error)
			os.Exit(1)
		}

		if command[0] == "exit" {
			break
		} else if command[0] == "echo" {
			if len(command) == 1 {
				fmt.Println()
			} else {
				for i := 1; i < len(command); i++ {
					fmt.Print(command[i] + " ")
				}
				fmt.Println()
			}
		} else if command[0] == "type" {

			if len(command) == 1 {
				fmt.Print()
			} else {
				for i := 1; i < len(command); i++ {
					if command[i] == "echo" || command[i] == "exit" || command[i] == "type" {
						fmt.Println(command[i] + " is a shell builtin")
					} else {
						foundExec := false
						path := os.Getenv("PATH")
						pathSlice := strings.Split(path, ":")
						var fullPath string

						for j := 0; j < len(pathSlice); j++ {

							if pathSlice[j][len(pathSlice[j])-1] != '/' {
								fullPath = pathSlice[j] + "/" + command[i]
							} else {
								fullPath = pathSlice[j] + command[i]
							}

							fileInfo, error := os.Stat(fullPath)
							if error == nil {
								mode := fileInfo.Mode()
								if mode&0b001001001 != 0 { // mode is stored as rwxrwxrwx
									fmt.Printf("%s is %s\n", command[i], fullPath)
									foundExec = true
									break
								}
							}

						}
						if !foundExec {
							fmt.Println(command[i] + ": not found")
						}
					}
				}
			}
		} else {
			fmt.Println(command[0] + ": command not found")
		}
	}
}
