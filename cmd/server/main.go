package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type projectContext struct {
	projectName string
	projectType string
}

var serverBuildDir = "build_directory"

func startServer(port string) {

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("TCP Server listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClient(conn)
	}
}

func handleFileMonitoring(reader *bufio.Reader, ctx projectContext) {

	for {
		// Read the length of the relativePath
		lengthStr, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading length of relative Path:", err)
			break
		}

		length, err := strconv.Atoi(strings.TrimSpace(lengthStr))
		if err != nil {
			log.Println("Error converting relative path length to integer", err)
			break
		}

		// Read the relativePath
		relativePath := make([]byte, length)
		_, err = io.ReadFull(reader, relativePath)
		if err != nil {
			log.Println("Error reading relativePath:", err)
			break
		}

		// Read the length of the content
		lengthStr, err = reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading content length string:", err)
			break
		}

		// convert content lengthStr to integer
		length, err = strconv.Atoi(strings.TrimSpace(lengthStr))
		if err != nil {
			log.Println("Error converting content length to integer", err)
			break
		}

		// Read the content
		content := make([]byte, length)
		_, err = io.ReadFull(reader, content)
		if err != nil {
			log.Println("Error reading file content", err)
			break
		}

		filePath := filepath.Join(serverBuildDir, ctx.projectName, string(relativePath))

		err = writeFile(filePath, content)
		if err != nil {
			fmt.Println("Error Writing File:", err)
			break
		}
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	initLine, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read initial info %v", err)
		return
	}

	parts := strings.Split(strings.TrimSpace(initLine), " ")
	if len(parts) < 3 || parts[0] != "INIT" {
		log.Println("Invalid initial info received")
		return
	}

	projectName := parts[1]
	projectType := parts[2]

	ctx := projectContext{
		projectName: projectName,
		projectType: projectType,
	}

	log.Printf("Setting up project context:\n\tProjectName: %s\n\tProjectType: %s \n", ctx.projectName, ctx.projectType)

	handleFileMonitoring(reader, ctx)
}

func writeFile(path string, content []byte) error {

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func main() {

	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	startServer(*port)
}
