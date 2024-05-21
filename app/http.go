package main

import (
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
        server.SetupFileSystem(directory)
    }
    server.RegisterHandler("get", "/echo/{str}", func(request *Request, response *Response, context *Context) error {
        response.SetStatusCode(200)
        response.SetHeader("Content-Type", "text/plain")
        param := context.params["str"]
        response.SetBody([]byte(param))
        return nil
    })
    server.RegisterHandler("get", "/user-agent", func(request *Request, response *Response, context *Context) error {
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
