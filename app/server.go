package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	var response string
	path := request.URL.Path

	switch {
	case path == "/":
		response = "HTTP/1.1 200 OK\r\n\r\n"

	case path == "/user-agent":
		userAgent := request.UserAgent()
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)

	case strings.HasPrefix(path, "/echo/"):
		echo := strings.TrimPrefix(path, "/echo/")
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echo), echo)
		if request.Method == "GET" {
			values, ok := request.Header["Accept-Encoding"]
			if ok && strings.Contains(strings.Join(values, ","), "gzip") {
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				w.Write([]byte(echo))
				w.Close()
				body := buf.String()
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Length:%d\r\n\r\n%s", len(body), body)
			} else {
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echo), echo)
			}
		}

	case strings.HasPrefix(path, "/files/"):
		if request.Method == "GET" {
			dir := os.Args[2]
			fileName := strings.TrimPrefix(path, "/files/")
			file, err := os.ReadFile(dir + fileName)
			if err != nil || len(file) == 0 {
				response = "HTTP/1.1 404 Not Found\r\n\r\n"
				break
			}
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(file), file)
		}
		if request.Method == "POST" {
			// Read the body of the POST request
			dir := os.Args[2]
			fileName := strings.TrimPrefix(path, "/files/")
			body, err := io.ReadAll(io.LimitReader(request.Body, 100))
			if err != nil {
				fmt.Println("Error reading body:", err)
				response = "HTTP/1.1 400 Bad Request\r\n\r\nFailed to read request body"
				break
			}

			// Write the body content to a file
			err = os.WriteFile(dir+fileName, body, 0644)
			if err != nil {
				fmt.Println("Error writing file:", err)
				response = "HTTP/1.1 500 Internal Server Error\r\n\r\nFailed to write file"
				break
			}
			// Success response
			response = "HTTP/1.1 201 Created\r\n\r\n"
		}

	default:
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}

}

func main() {
	fmt.Println("Logs from your program will appear here!")
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
