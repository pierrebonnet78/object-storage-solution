package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
    "strings"
	"path/filepath"
	"time"
	"github.com/gorilla/websocket"
)


func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seed := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(seed)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[randGen.Intn(len(charset))]
	}
	return string(b)
}

var FolderName string 
var BucketName string
var fileName string
func websocketHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade:", err)
        return
    }
    defer conn.Close()

    tempDir, err := ioutil.TempDir("", "uploads")
    if err != nil {
        log.Println("Error creating temporary directory:", err)
        return
    }
    defer os.RemoveAll(tempDir) 

    combinedFileName := "combined_file.txt"
    combinedFilePath := filepath.Join(tempDir, combinedFileName)
    combinedFile, err := os.Create(combinedFilePath)
    if err != nil {
        log.Println("Error creating combined file:", err)
        return
    }
    defer combinedFile.Close()

    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            if err != io.EOF {
                log.Println("Read:", err)
            }
            break
        }

        if messageType == websocket.TextMessage {
            
            received := string(message)
            log.Println("Received:", received)
            bucketName := strings.Split(received, "/")[0]
            fileName = strings.Split(received, "/")[1]
            log.Println("Bucket", bucketName)
            FolderName := filepath.Base(received)
            log.Println("filename", fileName)
            log.Println("Folder", FolderName)
            // concatening the folder name with the filename
            FolderName = FolderName+"/"+fileName
            log.Println("Folder Name:", FolderName)

            ext := filepath.Ext(FolderName)

            newFileName := randomString(10) + ext

            filePath := filepath.Join(tempDir, newFileName)
            file, err := os.Create(filePath)
            if err != nil {
                log.Println("Error creating file:", err)
                break
            }
            _, err = combinedFile.WriteString(fileName + ":\n")
            if err != nil {
                log.Println("Error writing original filename to combined file:", err)
                file.Close()
                os.Remove(filePath)
                break
            }

            err = conn.WriteMessage(websocket.TextMessage, []byte("ready"))
            if err != nil {
                log.Println("Error sending ready message:", err)
                file.Close()
                os.Remove(filePath)
                break
            }

            messageType, content, err := conn.ReadMessage()
            if err != nil {
                log.Println("Error reading file content:", err)
                file.Close()
                os.Remove(filePath)
                break
            }
            if messageType != websocket.BinaryMessage {
                log.Println("Invalid message type, expected BinaryMessage")
                file.Close()
                os.Remove(filePath)
                break
            }


            _, err = file.Write(content)
            if err != nil {
                log.Println("Error writing file content:", err)
                file.Close()
                os.Remove(filePath)
                break
            }

            err = uploadToMinioFolder(filePath, FolderName,bucketName)
            if err != nil {
                log.Println("Error uploading file to Minio:", err)
            }

            err = file.Close()
            if err != nil {
                log.Println("Error closing file:", err)
                os.Remove(filePath)
                break
            }

            err = os.Remove(filePath)
            if err != nil {
                log.Println("Error deleting file:", err)
            }
        }
      
    }

    log.Println("File upload completed")
}