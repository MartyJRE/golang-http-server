package main

import (
    "fmt"
    "io/fs"
    "log"
    "net"
    "os"
    "regexp"
    "strings"
)

const (
    BufferSize = 1024
)

type Server struct {
    protocol    string
    address     string
    port        uint32
    fullAddress string
    listener    net.Listener
    handlers    map[uint32]Handler
    patterns    map[uint32]*regexp.Regexp
    maxId       uint32
    regex       *regexp.Regexp
}

func (server *Server) HandleConnection(connection net.Conn) {
    buffer := make([]byte, BufferSize)
    _, readError := connection.Read(buffer)
    if readError != nil {
        log.Fatalln("Error reading data: ", readError.Error())
    }

    request := NewRequest(buffer)

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
        pattern := server.patterns[handlerId]
        if pattern == nil {
            log.Fatalln("No such pattern exists!", pattern)
        }
        if pattern.Match([]byte(path)) {
            context, contextErr := ContextFromRequest(&request)
            if contextErr != nil {
                log.Fatalln(fmt.Sprintf("Error creating context: %s", contextErr.Error()))
            }
            context.PopulateParams(pattern)
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

func (server *Server) RegisterHandler(path string, handler Handler) {
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
    server.patterns[id] = regex
    server.handlers[id] = handler
    server.maxId = id
}

func (server *Server) SetupFileSystem(directory string) {
    server.RegisterHandler("/files/{filename}", func(request *Request, response *Response, context *Context) error {
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
        patterns: map[uint32]*regexp.Regexp{
            0: regexp.MustCompile("^/$"),
        },
        regex: regexp.MustCompile("\\{([a-z]\\w+)}"),
    }
}
