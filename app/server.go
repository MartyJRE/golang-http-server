package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io/fs"
    "log"
    "net"
    "os"
    "regexp"
    "strings"
    "time"
)

type Server struct {
    protocol    string
    address     string
    port        uint32
    fullAddress string
    listener    net.Listener
    handlers    map[uint32]Handler
    patterns    map[uint32]Pattern
    maxId       uint32
    regex       *regexp.Regexp
}

func (server *Server) HandleConnection(connection net.Conn) {
    reader := bufio.NewReader(connection)
    buffer := make([]byte, reader.Size())
    _, readError := reader.Read(buffer)
    if readError != nil {
        log.Fatalln("Error reading data: ", readError.Error())
    }

    request := NewRequest(bytes.TrimRight(buffer, string([]byte{0})))

    method, methodError := request.Method()
    if methodError != nil {
        log.Fatalln(fmt.Sprintf("Error parsing method: %s", methodError.Error()))
    }
    path, pathError := request.Path()
    if pathError != nil {
        log.Fatalln(fmt.Sprintf("Error parsing path: %s", pathError.Error()))
    }
    protocol, protocolError := request.Protocol()
    if protocolError != nil {
        log.Fatalln(fmt.Sprintf("Error parsing protocol: %s", protocolError.Error()))
    }

    log.Println(fmt.Sprintf("[HTTP/%.1f] %s - %s", protocol, method, path))

    for handlerId, handler := range server.handlers {
        if pattern, ok := server.patterns[handlerId]; !ok {
            log.Fatalln("No such pattern exists!", pattern)
        } else if strings.ToLower(method) == pattern.method && pattern.regex.Match([]byte(path)) {
            context, contextErr := ContextFromRequest(&request)
            if contextErr != nil {
                log.Fatalln(fmt.Sprintf("Error creating context: %s", contextErr.Error()))
            }
            context.PopulateParams(pattern.regex)
            response := NewResponse()
            handlerError := handler(&request, &response, &context)
            if handlerError != nil {
                errorResponse := NewResponse()
                errorResponse.SetStatusCode(500)
                log.Println(fmt.Sprintf("[HTTP/%.1f] %s - %s : %d", protocol, method, path, errorResponse.statusCode))
                _, writeError := connection.Write(errorResponse.Build())
                if writeError != nil {
                    log.Fatalln("Failed to write response: ", writeError.Error())
                }
                return
            } else {
                _, writeError := connection.Write(response.Build())
                log.Println(fmt.Sprintf("[HTTP/%.1f] %s - %s : %d", protocol, method, path, response.statusCode))
                if writeError != nil {
                    log.Fatalln("Failed to write response: ", writeError.Error())
                }
                return
            }
        }
    }

    log.Printf("Path %s doesn't exist!\n", path)
    response := NewResponse()
    response.SetStatusCode(404)
    _, writeError := connection.Write(response.Build())
    if writeError != nil {
        log.Fatalln("Failed to write response: ", writeError.Error())
    }
}

func (server *Server) Start() {
    listener, bindError := net.Listen(server.protocol, server.fullAddress)
    if bindError != nil {
        log.Fatalf("Failed to bind to port %d\n", server.port)
    }
    server.listener = listener
    for {
        connection, acceptError := server.listener.Accept()
        if acceptError != nil {
            log.Fatalln("Error accepting connection: ", acceptError.Error())
        }
        go server.HandleConnection(connection)
    }
}

func (server *Server) RegisterHandler(method string, path string, handler Handler) {
    found := server.regex.FindAllString(path, -1)
    replacedPath := "^" + path + "$"
    for _, str := range found {
        str = strings.TrimRight(strings.TrimLeft(str, "{"), "}")
        this := fmt.Sprintf("%s%s}", "{", str)
        that := fmt.Sprintf("(?P<%s>\\w+)", str)
        replacedPath = strings.Replace(replacedPath, this, that, 1)
    }
    regex := regexp.MustCompile(replacedPath)
    id := server.maxId + 1
    server.patterns[id] = NewPattern(regex, method)
    server.handlers[id] = func(request *Request, response *Response, context *Context) error {
        log.Printf("Starting execution of handler %d\n", id)
        start := time.Now()
        result := handler(request, response, context)
        took := time.Since(start)
        log.Printf("Finished execution of handler %d - took: %s\n", id, DurationToTook(&took))
        return result
    }
    server.maxId = id
}

func DurationToTook(dur *time.Duration) string {
    return fmt.Sprintf(
        "%2.f s %d ms %d µs %d ns",
        dur.Seconds(),
        dur.Milliseconds(),
        dur.Microseconds(),
        dur.Nanoseconds(),
    )
}

func (server *Server) SetupFileSystem(directory string) {
    server.RegisterHandler("get", "/files/{filename}", func(request *Request, response *Response, context *Context) error {
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
    server.RegisterHandler("post", "/files/{filename}", func(request *Request, response *Response, context *Context) error {
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
        log.Println(request.Body())
        _, err = file.Write(request.Body())
        if err != nil {
            return err
        }
        response.SetStatusCode(201)
        return nil
    })
}

func NewServer(protocol string, address string, port uint32) Server {
    fullAddress := fmt.Sprintf("%s:%d", address, port)

    return Server{
        protocol:    protocol,
        address:     address,
        port:        port,
        fullAddress: fullAddress,
        handlers: map[uint32]Handler{
            0: func(req *Request, res *Response, _ *Context) error {
                res.SetStatusCode(200)
                return nil
            },
        },
        patterns: map[uint32]Pattern{
            0: Pattern{
                regex:  regexp.MustCompile("^/$"),
                method: "get",
            },
        },
        regex: regexp.MustCompile("\\{([a-z]\\w+)}"),
    }
}
