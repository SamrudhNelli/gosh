package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"sync"
	"strconv"
	"path/filepath"

	"github.com/chzyer/readline"
)

var builtin = map[string]bool{
	"echo" : true,
	"exit" : true, 
	"pwd" : true, 
	"type" : true, 
	"cd" : true,
	"history" : true,
}

var currHistory = make([]string, 0, 500)
var currHistoryInit int = 1

type bellCompleter struct {
	readline.AutoCompleter
	lastTabTime    time.Time
	secondTabPress bool
}

func initCompleters() []readline.PrefixCompleterInterface {
	uniqueStrings := make(map[string]bool, 5000)
	for key, val := range builtin {
		if val {
			uniqueStrings[key] = true
		}
	}
	path := os.Getenv("PATH")
	pathSlice := strings.Split(path, ":")
	for i := 0; i < len(pathSlice); i++ {
		entries, err := os.ReadDir(pathSlice[i])
		if err != nil {
			continue
		}
		for j := 0; j < len(entries); j++ {
			if entries[j].IsDir() || uniqueStrings[entries[j].Name()] {
				continue
			}
			fileInfo, err := entries[j].Info()
			if err != nil {
				continue
			}
			if fileInfo.Mode()&0111 != 0 {
				uniqueStrings[entries[j].Name()] = true
			}
		}
	}
	sortedStrings := make([]string, 0, len(uniqueStrings))
	for str, _ := range uniqueStrings {
		sortedStrings = append(sortedStrings, str)
	}
	slices.Sort(sortedStrings)
	options := make([]readline.PrefixCompleterInterface, 0, len(sortedStrings))
	for _, str := range sortedStrings {
		options = append(options, readline.PcItem(str))
	}
	return options
}

func (bc *bellCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	newLine, length = bc.AutoCompleter.Do(line, pos)
	if len(newLine) == 0 {
		fmt.Print("\x07")
		return newLine, length
	} else if len(newLine) == 1 {
		return newLine, length
	}

	longestCommonPrefix := newLine[0]
	for _, val := range newLine[1:] {
		minLen := min(len(longestCommonPrefix), len(val))
		i := 0
		for ; i < minLen; i++ {
			if longestCommonPrefix[i] != val[i] {
				break
			}
		}
		longestCommonPrefix = longestCommonPrefix[:i]
	}
	if len(longestCommonPrefix) > 0 {
		return [][]rune{longestCommonPrefix}, length
	}

	curTime := time.Now()
	if bc.secondTabPress && curTime.Sub(bc.lastTabTime) < 5*time.Second {
		fmt.Println("")
		for i, _ := range newLine {
			fullString := string(line) + string(newLine[i])
			fmt.Print(fullString)
			if i < len(newLine)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Print("\n")
		fmt.Printf("$ %s", string(line))
		bc.secondTabPress = false
		return nil, length
	}
	bc.lastTabTime = curTime
	bc.secondTabPress = true
	fmt.Print("\x07")
	return nil, length
}

func Echo(command []string) (print string) {
	size, flag := checkRedirectRequest(command)
	if size == -1 {
		size = len(command)
	}

	if size == 1 {
		print = "\n"
	} else {
		print = strings.Join(command[1:size], " ")
		print += "\n"
	}
	if flag != -1 {
		print := redirectEchoToFile(command, size, print, flag)
		return print
	}
	return
}

func Type(command []string) (print string) {
	if len(command) == 1 {
		print = ""
	} else {
		for i := 1; i < len(command); i++ {
			if builtin[command[i]] {
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

func Pwd() string {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s\n", path)
}

func Cd(command []string) string {
	if len(command) == 1 || command[1] == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		err = os.Chdir(home)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fileInfo, err := os.Stat(command[1])
		if err == nil && fileInfo.IsDir() {
			err = os.Chdir(command[1])
			if err != nil {
				log.Fatal(err)
			}
		} else {
			return fmt.Sprintf("cd: %s: No such file or directory\n", command[1])
		}
	}
	return ""
}

func getHistoryPath() string {
    home, err := os.UserHomeDir()
	if err != nil {
		fmt.Print("Error accessing home directory! : ")
		log.Fatal(err)
	}
    filePath := home + "/.gosh_history"
	if envPath := os.Getenv("HISTFILE"); envPath != "" {
		return envPath
    }
	return filePath
}

func saveToHistory() {
	historyPath := getHistoryPath()
	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	for _, val := range currHistory {
		val += "\n"
		data := []byte(val)
		_, err = file.Write(data)
		if err != nil {
			file.Close()
			return
		}
	}
}

func appendToCurrHistory(rawCommand string) {
	currHistory = append(currHistory, rawCommand)
}

func History(command []string) string {
	if len(command) > 2 && command[1] == "-r" {
		if strings.HasPrefix(command[2], "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			if command[2] == "~" {
				command[2] = home
			} else if strings.HasPrefix(command[2], "~/") {
				command[2] = filepath.Join(home, command[2][2:])
			}
    	}
		absPath, err := filepath.Abs(command[2])
		if err != nil {
			return ""
		}
		command[2] = string(absPath)
		content, err := os.ReadFile(command[2])
		if err != nil {
			return ""
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if line != "" {
				currHistory = append(currHistory, line)
			}
		}
		return ""
	}
	return history(command)
}

func history(command []string) string {

	content, err := os.ReadFile(getHistoryPath())
	if err != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	var output strings.Builder
	var maxLimit int = len(lines) + len(currHistory)

	if len(command) > 1 {
		if val ,err := strconv.Atoi(command[1]); err == nil {
			maxLimit = val
		} else if command[1] == "-w" || command[1] == "-a" {
			return editHistoryFile(command)
		} 
	}

	if len(currHistory) < maxLimit {
		for i := max(0, len(lines) + len(currHistory) - maxLimit - 1); i < len(lines); i++ {
			if lines[i] != "" {
				fmt.Fprintf(&output, "%5d  %s\n", i+1, lines[i])
			}
		}
	}
	for i := max(0, len(currHistory) - maxLimit); i < len(currHistory); i++ {
		if currHistory[i] != "" {
			fmt.Fprintf(&output, "%5d  %s\n", currHistoryInit + i, currHistory[i])
		}
	}
	return output.String()
}

func editHistoryFile(command []string) string {
	if len(command) < 3 {
		return ""
	}

	if strings.HasPrefix(command[2], "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		if command[2] == "~" {
			command[2] = home
		} else if strings.HasPrefix(command[2], "~/") {
			command[2] = filepath.Join(home, command[2][2:])
		}
	}
	absPath, err := filepath.Abs(command[2])
	if err != nil {
		return ""
	}
	command[2] = string(absPath)

	if command[1] == "-w" {
		file, err := os.OpenFile(command[2], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return ""
		}
		defer file.Close()
		for _, line := range currHistory {
			if line != "" {
				if _, err := file.WriteString(line + "\n"); err != nil {
					return ""
				}
			}
		}
	} else {
		file, err := os.OpenFile(command[2], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return ""
		}
		defer file.Close()
		for _, line := range currHistory {
			if _, err := file.WriteString(line + "\n"); err != nil {
				return ""
			}
		}
		currHistoryInit = currHistoryInit + len(currHistory)
		saveToHistory()
		currHistory = make([]string, 0, 500)
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

		fileInfo, err := os.Stat(fullPath)
		if err == nil {
			mode := fileInfo.Mode()
			if mode&0b001001001 != 0 { // mode is stored as rwxrwxrwx
				return true, fullPath
			}
		}
	}
	return false, ""
}

func isExec(command []string) bool {
	foundExec, _ := findExec(command[0])
	return foundExec
}

func returnExec(command []string) (cmd *exec.Cmd, outFile *os.File) {
	index, flag := checkRedirectRequest(command)
	var err error
	if flag == 1 {
		outFile, err = os.Create(command[index+1])
		if err != nil {
			log.Fatal(err)
		}
		cmd = exec.Command(command[0], command[1:index]...)
		cmd.Stdout = outFile
		cmd.Stderr = os.Stderr
	} else if flag == 2 {
		outFile, err = os.Create(command[index+1])
		if err != nil {
			log.Fatal(err)
		}
		cmd = exec.Command(command[0], command[1:index]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = outFile
	} else if flag == 3 {
		outFile, err = os.OpenFile(command[index+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		cmd = exec.Command(command[0], command[1:index]...)
		cmd.Stdout = outFile
		cmd.Stderr = os.Stderr
	} else if flag == 4 {
		outFile, err = os.OpenFile(command[index+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		cmd = exec.Command(command[0], command[1:index]...)
		cmd.Stderr = outFile
		cmd.Stdout = os.Stdout
	} else {
		cmd = exec.Command(command[0], command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd, nil
	}
	return
}

func RunExec(command []string) {
	cmd, outFile := returnExec(command)
	if outFile != nil {
		defer outFile.Close()
	}
	err := cmd.Run()
	// if err != nil {
	// 	// _, fullPath := findExec(command[0])
	// 	// fmt.Printf("Something went wrong! Could not execute %s\n", fullPath)
	// 	// log.Fatal(err)
	// }
	var _ = err
}

func checkRedirectRequest(command []string) (int, int) {
	index := slices.IndexFunc(command, func(c string) bool {
		return c == "1>" || c == ">" || c == "2>" || c == "1>>" || c == ">>" || c == "2>>"
	})
	if index == -1 {
		return index, -1
	} else if command[index] == "2>" {
		return index, 2
	} else if command[index] == "2>>" {
		return index, 4
	} else if command[index] == "1>" || command[index] == ">" {
		return index, 1
	} else {
		return index, 3
	}
}

func redirectEchoToFile(command []string, index int, print string, flag int) string {
	if len(command) == index+1 {
		fmt.Println("No output file specified!!")
		return ""
	}

	switch flag {
	case 1:
		data := []byte(print + strings.Join(command[min(index+2, len(command)):], " "))
		err := os.WriteFile(command[index+1], data, 0644)
		if err != nil {
			log.Fatal(err)
		}
	case 2:
		data := []byte("")
		err := os.WriteFile(command[index+1], data, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return print
	case 3:
		data := []byte(print + strings.Join(command[min(index+2, len(command)):], " "))
		file, err := os.OpenFile(command[index+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		_, err = file.Write(data)
		if err != nil {
			file.Close()
			log.Fatal(err)
		}
	case 4:
		data := []byte("")
		file, err := os.OpenFile(command[index+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		_, err = file.Write(data)
		if err != nil {
			file.Close()
			log.Fatal(err)
		}
		return print
	}
	return ""
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
			switch rawCommand[i] {
			case '\'', '"':
				startingQuote = rawCommand[i]
				insideWord = true
			case ' ', '\n', '\t', '\r', '\f', '\v':
				if temp != "" {
					command = append(command, temp)
					temp = ""
				}
			default:
				temp += string(rawCommand[i])
			}
		}
	}
	if temp != "" {
		command = append(command, temp)
	}
	return
}

func ExecutePipes(command []string) {
	commands := splitPipes(command)
	var cmds []*exec.Cmd
	var prevPipeReader *os.File = nil
	var wg sync.WaitGroup

	for idx, val := range commands {
		var currPipeReader *os.File = nil
        var currPipeWriter *os.File = nil
		if idx < len(commands) - 1 {
            r, w, err := os.Pipe()
			if err != nil {
				log.Fatal(err)
			}
            currPipeReader = r
            currPipeWriter = w
        }
		shouldCloseWriter := false

		if builtin[val[0]] {
			wg.Add(1)
			go func(v []string, w *os.File) {
				defer wg.Done()
				builtinInPipe(v, w)
			}(val, currPipeWriter)
		} else {
			cmd := exec.Command(val[0], val[1:]...)

			if prevPipeReader != nil {
				cmd.Stdin = prevPipeReader
			} else {
				cmd.Stdin = os.Stdin
			}
			if currPipeWriter != nil {
				cmd.Stdout = currPipeWriter
				shouldCloseWriter = true
			} else {
				cmd.Stdout = os.Stdout
			}
			cmd.Stderr = os.Stderr
			cmd.Start()
			cmds = append(cmds, cmd)
		}

		if shouldCloseWriter && currPipeWriter != nil {
			currPipeWriter.Close()
		}
		if prevPipeReader != nil {
			prevPipeReader.Close()
		}        
		prevPipeReader = currPipeReader
	}

	for _, cmd := range cmds {
		cmd.Wait()
	}
	wg.Wait()
}

func builtinInPipe(command []string, out *os.File) {
	var cmdOutput string
	switch command[0] {
	case "echo" : cmdOutput = Echo(command)
	case "pwd" : cmdOutput = Pwd()
	case "cd" : cmdOutput = Cd(command)
	case "type" : cmdOutput = Type(command)
	case "history" : cmdOutput = History(command)
	default : log.Fatal("Internal builtin code broken!")
	}

	if out != nil {
		out.WriteString(cmdOutput)
		out.Close()
	} else {
		fmt.Print(cmdOutput)
	}
}

func splitPipes(command []string) (commands [][]string) {
	var commandSet []string
	for _, val := range command {
		if val == "|" {
			commands = append(commands, commandSet)
			commandSet = nil		
		} else {
			commandSet = append(commandSet, val)
		}
	}
	if commandSet != nil {
		commands = append(commands, commandSet)
	}
	return
}

func hasPipelines(command []string) bool {
	for _, val := range command {
		if val == "|" {
			return true
		}
	}
	return false
}

func Welcome() {
	cyan := "\033[36m"
    white := "\033[97m"
    reset := "\033[0m"

    fmt.Println(cyan + "\n                  .oooooo.      .oooooo.            oooo")
    fmt.Println("                 d8P'  `Y8b    d8P'  `Y8b           `888")
    fmt.Println("                888           888      888  .oooo.o  888 .oo.")
    fmt.Println("                888           888      888 d88(  \"8  888P\"Y88b")
    fmt.Println("                888     ooooo 888      888 `\"Y88b.   888   888")
    fmt.Println("                `88.    .88'  `88b    d88' o.  )88b  888   888")
    fmt.Println("                 `Y8bood8P'    `Y8bood8P'  8\"\"888P' o888o o888o" + reset)
    
    fmt.Println(white + "\n                           v1.0 (The Go Shell)\n" + reset)
    fmt.Println()
}

func main() {

	historyPath := getHistoryPath()
	completer := readline.NewPrefixCompleter(initCompleters()...)
	customCompleter := &bellCompleter{
		AutoCompleter: completer,
	}

	// Welcome()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "$ ",
		AutoComplete:      customCompleter,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	if content, err := os.ReadFile(historyPath); err == nil {
        lines := strings.Split(string(content), "\n")
        for _, line := range lines {
            if line != "" {
				currHistoryInit++
                rl.SaveHistory(line)
            }
        }
    }
	defer saveToHistory()

	for {

		rawCommand, err := rl.Readline()
		if err == readline.ErrInterrupt { // Ctrl + C
			if len(rawCommand) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF { // Ctrl + D and other errs
			break
		} else if err != nil { // Other rare errors! Something broke!!
			break
		}

		command := commandParser(rawCommand)

		if len(command) == 0 {
			continue
		}
		appendToCurrHistory(rawCommand)

		if command[0] == "exit" {
			break
		} else if hasPipelines(command) {
			ExecutePipes(command) 
		} else if command[0] == "echo" {
			fmt.Print(Echo(command))
		} else if command[0] == "type" {
			fmt.Print(Type(command))
		} else if command[0] == "pwd" {
			fmt.Print(Pwd())
		} else if command[0] == "cd" {
			fmt.Print(Cd(command))
		} else if command[0] == "history" {
			fmt.Print(History(command))
		} else if isExec(command) {
			RunExec(command)
		} else {
			fmt.Println(command[0] + ": command not found")
		}
	}
}
