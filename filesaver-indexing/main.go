package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/template"
)

var storageServerPicker int = 0

var filesIndexing = []FileLocation{}

type FileLocation struct {
	FileName      string
	FileSize      int64
	StorageServer StorageServer
}

type StorageServer struct {
	Host string
	Port string
}

var storageServers = []StorageServer{
	{Host: "filesaverstorage1", Port: "33333"},
	{Host: "filesaverstorage2", Port: "33332"},
}
var templates = template.Must(template.ParseFiles("public/upload.html"))

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/overview", overviewHandler)
	http.HandleFunc("/readFile", readHandler)
	http.HandleFunc("/deletefile", deleteFileHandler)

	http.ListenAndServe(":8080", nil)
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("filename")
	if fileName == "" {
		http.Error(w, "delete file unsuccessful", 400)
		return
	}
	idxToBeRemoved := -1
	for i, v := range filesIndexing {
		if v.FileName == fileName {
			idxToBeRemoved = i
		}
	}

	if idxToBeRemoved != -1 {

		go deleteFileFromStorageServer(filesIndexing[idxToBeRemoved])

		filesIndexing[idxToBeRemoved] = filesIndexing[len(filesIndexing)-1]
		filesIndexing[len(filesIndexing)-1] = FileLocation{}
		filesIndexing = filesIndexing[:len(filesIndexing)-1]

		fmt.Fprintf(w, "File deleted successful")
	} else {
		fmt.Fprintf(w, "File not found")
	}
	return
}

func display(w http.ResponseWriter, page string, data interface{}) {
	templates.ExecuteTemplate(w, page+".html", data)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	file, handler, _ := r.FormFile("myFile")

	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	err := sendFileToStorageServer(file, handler.Size, handler.Filename)

	if err != "" {
		http.Error(w, err, 500)
		return
	}

	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		display(w, "upload", nil)
	case "POST":
		uploadFile(w, r)
	}
}

func overviewHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(filesIndexing)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprint(w, string(b))
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("filename")
	if fileName == "" {
		http.Error(w, "bad filename", 400)
		return
	}
	idxFound := -1
	for i, v := range filesIndexing {
		if v.FileName == fileName {
			idxFound = i
		}
	}

	if idxFound != -1 {
		fileLocation := filesIndexing[idxFound]

		readFileFromStorageServer(fileLocation)
		// defer file.Close()
		file, err := os.Open("FileToReturn.txt")

		if err != nil {
			fmt.Println("error opening file to return")
		}

		fileHeader := make([]byte, 512)
		file.Read(fileHeader)
		fileType := http.DetectContentType(fileHeader)

		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.Header().Set("Content-Control", "private, no-transform, no-store, must-revalidate")
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName+".txt")
		w.Header().Set("Content-Type", fileType)
		w.Header().Set("Content-Length", strconv.FormatInt(fileLocation.FileSize, 10))

		file.Seek(0, 0)
		io.Copy(w, file)
		fmt.Fprintf(w, "File deleted successful")
	} else {
		fmt.Fprintf(w, "File not found")
	}
	return

}

func sendFileToStorageServer(file multipart.File, size int64, fileName string) string {

	storageServer := storageServers[storageServerPicker]
	conn, err := net.Dial("tcp", storageServer.Host+":"+storageServer.Port)
	if err != nil {
		fmt.Println("Error starting connection: ", err)
		return "Internal error"
	}
	defer conn.Close()

	fmt.Println("A client has connected!")
	defer conn.Close()
	fmt.Println("Sending filename and filesize!")

	fileSize := fillString(strconv.FormatInt(size, 10), 10)
	fileNameFormatted := fillString(fileName, 64)
	conn.Write([]byte(fillString("CREATE", 10)))
	conn.Write([]byte(fileSize))
	conn.Write([]byte(fileNameFormatted))
	sendBuffer := make([]byte, 1024)
	fmt.Println("Start sending file!")
	for {
		_, err = file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		conn.Write(sendBuffer)
	}
	fmt.Println("File has been sent, closing connection!")
	fl := FileLocation{
		FileName:      fileName,
		FileSize:      size,
		StorageServer: storageServer,
	}
	filesIndexing = append(filesIndexing, fl)
	storageServerPicker = (storageServerPicker + 1) % len(storageServers)
	return ""
}

func deleteFileFromStorageServer(fileLocation FileLocation) {
	conn, err := net.Dial("tcp", fileLocation.StorageServer.Host+":"+fileLocation.StorageServer.Port)
	if err != nil {
		fmt.Println("Error starting connection: ", err)
	}
	defer conn.Close()

	fmt.Println("A client has connected!")
	defer conn.Close()
	fmt.Println("Sending filename and filesize!")

	// fileSize := fillString(strconv.FormatInt(size, 10), 10)
	fileNameFormatted := fillString(fileLocation.FileName, 64)
	conn.Write([]byte(fillString("DELETE", 10)))
	conn.Write([]byte(fileNameFormatted))
	return
}

func readFileFromStorageServer(fileLocation FileLocation) {
	conn, err := net.Dial("tcp", fileLocation.StorageServer.Host+":"+fileLocation.StorageServer.Port)
	if err != nil {
		fmt.Println("Error starting connection: ", err)
	}
	defer conn.Close()

	fmt.Println("A client has connected!")
	defer conn.Close()
	fmt.Println("Sending filename")

	fileNameFormatted := fillString(fileLocation.FileName, 64)
	conn.Write([]byte(fillString("READ", 10)))
	conn.Write([]byte(fileNameFormatted))
	var receivedBytes int64
	fileToReturn, err := os.Create("FileToReturn.txt")
	if err != nil {
		panic(err)
	}
	defer fileToReturn.Close()
	for {
		if (fileLocation.FileSize - receivedBytes) < 1024 {
			io.CopyN(fileToReturn, conn, (fileLocation.FileSize - receivedBytes))
			conn.Read(make([]byte, (receivedBytes+1024)-fileLocation.FileSize))
			break
		}

		io.CopyN(fileToReturn, conn, 1024)
		receivedBytes += 1024
	}
}

func fillString(rt string, toLength int) string {
	for {
		lengtString := len(rt)
		if lengtString < toLength {
			rt = rt + ":"
			continue
		}
		break
	}
	return rt
}
