package main

import (
    "io/fs"
    "log"
    "os"
)

func main() {
    server := NewServer("tcp", "0.0.0.0", 4221)
    args := os.Args[1:]
    if len(args) != 0 && (len(args) != 2 || args[0] != "--directory") {
        log.Fatalf("Invalid arguments:\nUsage:\n  %s [--directory <path>]\n", os.Args[0])
    }
    if len(args) == 2 {
        directory := args[1]
        server.RegisterHandler("get", "/files/{filename}", "Get file", func(request *Request, response *Response, context *Context) error {
            filename := context.params["filename"]
            info, err := os.Stat(directory + filename)
            if os.IsNotExist(err) {
                return err
            }
            if info.IsDir() {
                response.SetStatusCode(404)
                return nil
            }
            file, err := os.OpenFile(directory+filename, os.O_RDONLY, fs.ModeTemporary)
            if err != nil {
                return err
            }
            defer func(file *os.File) {
                fileErr := file.Close()
                if fileErr != nil {
                    log.Fatalln("Failed to close a file", fileErr.Error())
                }
            }(file)
            data := make([]byte, info.Size())
            _, err = file.Read(data)
            if err != nil {
                return err
            }
            response.SetStatusCode(200)
            response.SetHeader("Content-Type", "application/octet-stream")
            response.SetBody(data)
            return nil
        })
        server.RegisterHandler("post", "/files/{filename}", "Create file", func(request *Request, response *Response, context *Context) error {
            filename := context.params["filename"]
            file, err := os.Create(directory + filename)
            if err != nil {
                return err
            }
            defer func(file *os.File) {
                fileErr := file.Close()
                if fileErr != nil {
                    log.Fatalln("Failed to close a file", fileErr.Error())
                }
            }(file)
            _, err = file.Write(request.Body())
            if err != nil {
                return err
            }
            response.SetStatusCode(201)
            return nil
        })
    }
    server.RegisterHandler("get", "/echo/{str}", "Echo", func(request *Request, response *Response, context *Context) error {
        response.SetStatusCode(200)
        response.SetHeader("Content-Type", "text/plain")
        param := context.params["str"]
        response.SetBody([]byte(param))
        return nil
    })
    server.RegisterHandler("get", "/user-agent", "User agent", func(request *Request, response *Response, context *Context) error {
        response.SetStatusCode(200)
        response.SetHeader("Content-Type", "text/plain")
        userAgent, err := request.GetHeader("User-Agent")
        if err != nil {
            return err
        }
        response.SetBody([]byte(userAgent))
        return nil
    })
    server.Start()
}
