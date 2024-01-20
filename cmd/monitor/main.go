package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

var serverConn net.Conn

func connectToServer(serverAddress string) (net.Conn, error) {

	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func sendFile(conn net.Conn, relativePath string, content []byte) error {

	// Send the length of the relativePath
	_, err := conn.Write([]byte(fmt.Sprintf("%d\n", len(relativePath))))
	if err != nil {
		return err
	}

	// Send the relative path
	_, err = conn.Write([]byte(relativePath))
	if err != nil {
		return err
	}

	// Send the length of the content
	contentLengthStr := fmt.Sprintf("%d\n", len(content))
	_, err = conn.Write([]byte(contentLengthStr))
	if err != nil {
		return err
	}

	// Send the content
	_, err = conn.Write(content)
	return err
}

func handleFileEvent(baseDir string, event fsnotify.Event) {

	content, err := os.ReadFile(event.Name)
	if err != nil {
		log.Println("Error reading file data:", err)
		return
	}

	relativePath, err := filepath.Rel(baseDir, event.Name)
	if err != nil {
		log.Println("Error getting relative path:", err)
		return
	}

	err = sendFile(serverConn, relativePath, content)
	if err != nil {
		log.Println("Error sending data to server:", err)
		// TODO: Handle reconnecting to server if necessary
	}
}

func sendProjectInitialization(conn net.Conn, projectName string, projectType string) error {

	initialInfo := fmt.Sprintf("INIT %s %s \n", projectName, projectType)

	_, err := conn.Write([]byte(initialInfo))
	return err
}

func main() {

	targetDir := flag.String("targetDir", "", "the target directory to monitor")
	flag.Parse()

	if *targetDir == "" {
		log.Fatal("targetDir is required")
	}

	// The project name will be the name of the directory the client is monitoring
	projectName := filepath.Base(*targetDir)
	projectType := "go"

	var err error
	serverConn, err = connectToServer("localhost:8080")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer serverConn.Close()

	err = sendProjectInitialization(serverConn, projectName, projectType)
	if err != nil {
		log.Fatalf("Failed to send initial information: %v", err)
	}

	// Sets up the file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				handleFileEvent(*targetDir, event)

				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(*targetDir)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
