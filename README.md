### Custom distributed file system

The file system consists of two applications:

- **filesaver-indexing** which acts as the client as well as handles the indexing of the files.
- **filesaver-storage** which acts as the storage servers for the file system.

### Actions: 

    http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/overview", overviewHandler)
	http.HandleFunc("/readFile", readHandler)
	http.HandleFunc("/deletefile", deleteFileHandler)

readFile requires "filename" query parameter:

        http://localhost/readFile?filename=name.txt

deletefile requires "filename" query parameter:

        http://localhost/deletefile?filename=name.txt

### How to get started

- cd to filesaver-indexing directory

        docker build -t filesaver-indexing .

- cd to filesaver-storage directory

        docker build -t filesaver-storage .

- docker-compose up

- You should now be able to upload a file on the endpoint http://localhost:8080/upload in the browser.
