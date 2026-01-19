package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"os/exec"
	"log"
)

func findExec(program string) (bool, string) {
	path := os.Getenv("PATH")
	pathSlice := strings.Split(path, ":")
	var fullPath string

	for j := 0; j < len(pathSlice); j++ {
		if pathSlice[j][len(pathSlice[j])-1] != '/' {
			fullPath = pathSlice[j] + "/" + program
		} else {
			fullPath = pathSlice[j] + program
		}

		fileInfo, error := os.Stat(fullPath)
		if error == nil {
			mode := fileInfo.Mode()
			if mode&0b001001001 != 0 { // mode is stored as rwxrwxrwx
				return true, fullPath
			}
		}
	}
	return false, ""
}

func main() {
	for {
		fmt.Print("$ ")
		rawCommand, error := bufio.NewReader(os.Stdin).ReadString('\n')
		command := strings.Fields(rawCommand)

		if error != nil {
			log.Fatal(error)
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
					if command[i] == "echo" || command[i] == "exit" || command[i] == "type" || command[i] == "pwd" {
						fmt.Println(command[i] + " is a shell builtin")
					} else {
						foundExec, fullPath := findExec(command[i])
						if foundExec {
							fmt.Printf("%s is %s\n", command[i], fullPath)
						} else {
							fmt.Println(command[i] + ": not found")
						}
					}
				}
			}
		} else if command[0] == "pwd" {
			path, error := os.Getwd()
			if error != nil {
				log.Fatal(error)
			}
			fmt.Println(path)
		} else if command[0] == "cd" {
			if len(command) == 1 || command[1] == "~" {
				home, error := os.UserHomeDir()
				if error != nil {
					log.Fatal(error)
				}
				error = os.Chdir(home)
				if error != nil {
					log.Fatal(error)
				}
			} else {
				fileInfo, error := os.Stat(command[1])
				if error == nil && fileInfo.IsDir() {
					error = os.Chdir(command[1])
					if error != nil {
						log.Fatal(error)
					}
				} else {
					fmt.Printf("cd: %s: No such file or directory\n", command[1])
				}
			}
		} else if foundExec, fullPath := findExec(command[0]); foundExec {
			cmd := exec.Command(command[0], command[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			error := cmd.Run()
			if error != nil {
				fmt.Printf("Something went wrong! Could not execute %s\n", fullPath)
				log.Fatal(error)
			}
		} else {
			fmt.Println(command[0] + ": command not found")
		}
	}
}
