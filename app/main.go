package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"os/exec"
	"log"
)

func Echo(command []string) (print string) {

	if len(command) == 1 {
		print = "\n"
	} else {
		for i := 1; i < len(command); i++ {
			print += fmt.Sprintf(command[i] + " ")
		}
		print += "\n"
	}
	return
}

func Type(command []string) (print string) {
	if len(command) == 1 {
		print = ""
	} else {
		for i := 1; i < len(command); i++ {
			if command[i] == "echo" || command[i] == "exit" || command[i] == "type" || command[i] == "pwd" {
				print += fmt.Sprintf("%s is a shell builtin\n", command[i])
			} else {
				foundExec, fullPath := findExec(command[i])
				if foundExec {
					print += fmt.Sprintf("%s is %s\n", command[i], fullPath)
				} else {
					print += fmt.Sprintf("%s: not found\n", command[i])
				}
			}
		}
	}
	return
}

func Pwd() (string) {
	path, error := os.Getwd()
	if error != nil {
		log.Fatal(error)
	}
	return fmt.Sprintf("%s\n", path)
}

func Cd(command []string) (string) {
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
			return fmt.Sprintf("cd: %s: No such file or directory\n", command[1])
		}
	}
	return ""
}

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

func isExec(command []string) (bool) {
	foundExec, _ := findExec(command[0])
	return foundExec
}

func runExec(command []string) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	error := cmd.Run()
	if error != nil {
		_, fullPath := findExec(command[0])
		fmt.Printf("Something went wrong! Could not execute %s\n", fullPath)
		log.Fatal(error)
	}
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
			fmt.Print(Echo(command))
		} else if command[0] == "type" {
			fmt.Print(Type(command))
		} else if command[0] == "pwd" {
			fmt.Print(Pwd())
		} else if command[0] == "cd" {
			fmt.Print(Cd(command))
		} else if isExec(command) {
			runExec(command)
		} else {
			fmt.Println(command[0] + ": command not found")
		}
	}
}
