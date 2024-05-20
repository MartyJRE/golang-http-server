package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "strings"
)

func main() {
    listener, err := net.Listen("tcp", "0.0.0.0:4221")

    if err != nil {
        fmt.Println("Failed to bind to port 4221")
        os.Exit(1)
    }

    connection, err := listener.Accept()
    if err != nil {
        fmt.Println("Error accepting connection: ", err.Error())
        os.Exit(1)
    }

    buffer := make([]byte, 1024)
    _, err = connection.Read(buffer)
    if err != nil {
        fmt.Println("Error reading data: ", err.Error())
        os.Exit(1)
    }

    message := string(buffer)
    splitLines := strings.Split(message, "\r\n")

    requestLine := splitLines[0]
    headers := splitLines[1 : len(splitLines)-1]
    body := splitLines[len(splitLines)-1]

    splitRequestLine := strings.Split(requestLine, string(' '))
    method := splitRequestLine[0]
    path := splitRequestLine[1]
    protocol := splitRequestLine[2]

    log.Println(fmt.Sprintf("[%s] %s - %s\n%s\n%v", protocol, method, path, headers, body))

    switch path {
    case "/":
        {
            writerBuffer := []byte("HTTP/1.1 200 OK\r\n\r\n")
            _, err = connection.Write(writerBuffer)
            if err != nil {
                fmt.Println("Failed to write response: ", err.Error())
                os.Exit(1)
            }
        }
    default:
        {
            writerBuffer := []byte("HTTP/1.1 404 Not Found\r\n\r\n")
            _, err = connection.Write(writerBuffer)
            if err != nil {
                fmt.Println("Failed to write response: ", err.Error())
                os.Exit(1)
            }
        }
    }
}
