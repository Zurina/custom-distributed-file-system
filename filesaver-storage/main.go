package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CONN_HOST  = "0.0.0.0"
	CONN_TYPE  = "tcp"
	BUFFERSIZE = 1024
)

func main() {
	ch := make(chan bool)
	go setupListenerToCreateFile(ch)
	<-ch
}

func setupListenerToCreateFile(ch chan bool) {
	CONN_PORT := os.Getenv("TCP_PORT")
	listener, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	fmt.Println("Connected to server, start receiving the file name and file size to create file")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		actionName := make([]byte, 10)
		conn.Read(actionName)
		action := strings.Trim(string(actionName), ":")
		if action == "CREATE" {
			go handleCreateFile(conn)
		} else if action == "DELETE" {
			go handleDeleteFile(conn)
		} else if action == "READ" {
			go handleReadFile(conn)
		}
	}
	ch <- true
}

func handleReadFile(conn net.Conn) {
	fmt.Println("HandleReadFile")
	bufferFileName := make([]byte, 64)
	conn.Read(bufferFileName)
	fileName := strings.Trim(string(bufferFileName), ":")

	splittedFileName := strings.Split(fileName, ".")
	onlyFileName := strings.Join(splittedFileName[:len(splittedFileName)-1], "")
	files, err := filepath.Glob(onlyFileName + "*")
	if err != nil {
		panic(err)
	}
	mergedFile, err := os.OpenFile("merged_file.txt", os.O_CREATE|os.O_WRONLY, 0644)
	// mergedFile, err := os.Create("merged_file.txt")
	if err != nil {
		log.Fatalln("failed to open outpout file:", err)
	}

	for _, f := range files {
		shard, err := os.Open(f)
		if err != nil {
			log.Fatalln("failed to open signed for reading:", err)
		}
		defer shard.Close()
		_, err = io.Copy(mergedFile, shard)
		if err != nil {
			log.Fatalln("failed to append zip file to output:", err)
		}
	}
	mergedFile.Close()
	sendBuffer := make([]byte, 1024)
	fmt.Println("Start sending file!")
	resultFile, err := os.OpenFile("merged_file.txt", os.O_RDONLY, 0644)
	if err != nil {
		log.Fatalln("failed to open result file")
		return
	}
	_, _ = resultFile.Read(sendBuffer)
	conn.Write(sendBuffer)
}

func handleDeleteFile(conn net.Conn) {
	fmt.Println("handleDeleteFile")
	bufferFileName := make([]byte, 64)
	conn.Read(bufferFileName)
	fileName := strings.Trim(string(bufferFileName), ":")

	splittedFileName := strings.Split(fileName, ".")
	onlyFileName := strings.Join(splittedFileName[:len(splittedFileName)-1], "")
	files, err := filepath.Glob(onlyFileName + "*")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			panic(err)
		}
		fmt.Println("Deleted", f)
	}
}

func handleCreateFile(conn net.Conn) {
	fmt.Println("handleCreateFile")
	bufferFileName := make([]byte, 64)
	bufferFileSize := make([]byte, 10)
	conn.Read(bufferFileSize)
	conn.Read(bufferFileName)

	fileSize, _ := strconv.ParseInt(strings.Trim(string(bufferFileSize), ":"), 10, 64)
	fileName := strings.Trim(string(bufferFileName), ":")

	var receivedBytes int64
	fileNameSuffix := 1
	splittedFileName := strings.Split(fileName, ".")
	onlyFileName := strings.Join(splittedFileName[:len(splittedFileName)-1], "")

	for {
		newFileShard, err := os.Create(onlyFileName + "_" + strconv.Itoa(fileNameSuffix) + ".txt")

		if err != nil {
			panic(err)
		}

		defer newFileShard.Close()

		if (fileSize - receivedBytes) < BUFFERSIZE {
			io.CopyN(newFileShard, conn, (fileSize - receivedBytes))
			conn.Read(make([]byte, (receivedBytes+BUFFERSIZE)-fileSize))
			break
		}

		io.CopyN(newFileShard, conn, BUFFERSIZE)
		receivedBytes += BUFFERSIZE
		fileNameSuffix++
	}
	fmt.Println("Received file completely!")
}
