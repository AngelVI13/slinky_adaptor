package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func getInput(reader io.Reader, ch chan string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		reply := scanner.Text()
		log.Printf("Reading from subprocess: %s", reply)
		ch <- reply
	}
}

func getEngineResponse(input string, stdin io.Writer, ch chan string) (output string) {
	if _, err := stdin.Write([]byte(input)); err != nil {
		log.Fatalf("Error writing to stdin: %s", err.Error())
	}

	stopped := false
	for stopped == false {
		select {
		case value := <-ch:
			fmt.Printf("Got input. %s\n", value)
			output += fmt.Sprintf("%s\n", value)
		case <-time.After(1 * time.Second):
			fmt.Println("Timed out, exiting.")
			stopped = true
		}
	}
	return output
}

func main() {
	cmd := exec.Command("./slinky.exe")
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if nil != err {
		log.Fatalf("Error obtaining stdin: %s", err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if nil != err {
		log.Fatalf("Error obtaining stdout: %s", err.Error())
	}

	ch := make(chan string)
	reader := bufio.NewReader(stdout)
	go getInput(reader, ch) // start coroutine that processes engine output

	if err := cmd.Start(); nil != err {
		log.Fatalf("Error starting program: %s, %s", cmd.Path, err.Error())
	}

	fmt.Println("---------------------------")
	fmt.Println(getEngineResponse("uci\n", stdin, ch))

	// Kill it:
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}
