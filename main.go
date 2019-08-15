package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	pb "./protos"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// server is used to implement AdapterServer.
type adapterServer struct {
	stdin io.Writer
	ch    chan string
}

// ExecuteEngineCommand implements AdapterServer
func (s *adapterServer) ExecuteEngineCommand(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	log.Printf("Received: %v", in.Text)
	response := getEngineResponse(in.Text, s.stdin, s.ch, in.Timeout)
	log.Printf("Sending back: %v", response)
	return &pb.Response{Text: response}, nil
}

func getInput(reader io.Reader, ch chan string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		reply := scanner.Text()
		log.Printf("Reading from subprocess: %s", reply)
		ch <- reply
	}
}

func getEngineResponse(input string, stdin io.Writer, ch chan string, waitSeconds int32) (output string) {
	if _, err := stdin.Write([]byte(input)); err != nil {
		log.Fatalf("Error writing to stdin: %s", err.Error())
	}

	stopped := false
	for stopped == false {
		select {
		case value := <-ch:
			fmt.Printf("Got input. %s\n", value)
			output += fmt.Sprintf("%s\n", value)
		case <-time.After(time.Duration(waitSeconds) * time.Second):
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

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := adapterServer{stdin: stdin, ch: ch}
	pb.RegisterAdapterServer(s, &server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// Todo add gravefull shutdown of engine process after command from gui is received
	// Kill it:
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}
