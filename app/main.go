package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"os/exec"
	"log"
	"slices"
)

func Echo(command []string) (print string) {
	size, flag := checkRedirectRequest(command)
	if size == -1 {
		size = len(command) 
	}

	if size == 1 {
		print = "\n"
	} else {
		for i := 1; i < size; i++ {
			print += fmt.Sprintf(command[i] + " ")
		}
		print += "\n"
	}
	if flag {
		redirectToFile(command, size, print)
		return ""
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
	index, flag := checkRedirectRequest(command)
	var cmd *exec.Cmd

	if flag {
		outFile, error := os.Create(command[index + 1])
		if error != nil {
			log.Fatal(error)
		}
		defer outFile.Close()
		cmd = exec.Command(command[0], command[1:index]...)
		cmd.Stdout = outFile
	} else {
		cmd = exec.Command(command[0], command[1:]...)
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr
	error := cmd.Run()
	// if error != nil {
	// 	// _, fullPath := findExec(command[0])
	// 	// fmt.Printf("Something went wrong! Could not execute %s\n", fullPath)
	// 	// log.Fatal(error)
	// }
	var _ = error
}

func checkRedirectRequest(command []string) (int, bool) {
	index := slices.IndexFunc(command, func(c string) bool {
		return c == "1>" || c == ">"
	})
	if index == -1 {
		return index, false
	} else {
		return index, true
	}
}

func redirectToFile(command []string, index int, print string) () {
	if len(command) == index + 1 {
		fmt.Println("No output file specified!!")
		return
	}
	data := []byte(print + strings.Join(command[min(index + 2, len(command)):], " "))
	error := os.WriteFile(command[index + 1], data, 0644)
	if error != nil {
		log.Fatal(error)
	}
	return
}

func commandParser(rawCommand string) (command []string) {
	var startingQuote byte
	var temp string = ""
	var insideWord bool = false
	for i := 0; i < len(rawCommand); i++ {
		if insideWord {
			if rawCommand[i] == startingQuote {
				insideWord = false
			} else {
				temp += string(rawCommand[i])
			}
		} else {
			if rawCommand[i] == '\'' || rawCommand[i] == '"' {
				startingQuote = rawCommand[i]
				insideWord = true
			} else if rawCommand[i] == ' ' || rawCommand[i] == '\n' || rawCommand[i] == '\t' || rawCommand[i] == '\r' || rawCommand[i] == '\f' || rawCommand[i] == '\v' {
				if temp != "" {
					command = append(command, temp)
					temp = ""
				}
			} else {
				temp += string(rawCommand[i])
			}
		}
	}
	if temp != "" {
		command = append(command, temp)
	}
	return
}


func main() {
	for {
		fmt.Print("$ ")
		rawCommand, error := bufio.NewReader(os.Stdin).ReadString('\n')
		command := commandParser(rawCommand)

		if error != nil {
			log.Fatal(error)
		}

		if len(command) == 0 {
			continue
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
