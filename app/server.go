package main

import (
    "bufio"
    "bytes"
    "fmt"
    "log"
    "net"
    "regexp"
    "strings"
    "time"

    "github.com/google/uuid"
)

type Server struct {
    protocol      string
    address       string
    port          uint32
    fullAddress   string
    listener      net.Listener
    handlers      map[uuid.UUID]Handler
    patterns      map[uuid.UUID]Pattern
    patternRegex  *regexp.Regexp
    encodingRegex *regexp.Regexp
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
            if server.HandleRequest(connection, request, pattern, handler, protocol, method, path) {
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

const GZip = "gzip"

var ValidEncodings = []string{GZip}

func ResolveValidEncodings(encodingStr string) []string {
    valid := make([]string, 0)
    encodings := strings.Split(encodingStr, ",")
    for _, v := range ValidEncodings {
        for _, encoding := range encodings {
            enc := strings.TrimSpace(encoding)
            if v == enc {
                valid = append(valid, v)
            }
        }
    }
    return valid
}

func (server *Server) HandleRequest(connection net.Conn, request Request, pattern Pattern, handler Handler, protocol float32, method string, path string) bool {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        log.Printf("[HTTP/%.1f] %s - %s - took: %s", protocol, method, path, DurationToTook(&duration))
    }()
    context, contextErr := ContextFromRequest(&request)
    if contextErr != nil {
        log.Fatalln(fmt.Sprintf("Error creating context: %s", contextErr.Error()))
    }
    context.PopulateParams(pattern.regex)
    response := NewResponse()
    handlerError := handler.handler(&request, &response, &context)
    if acceptEncoding, err := request.GetHeader("Accept-Encoding"); err == nil {
        encodings := ResolveValidEncodings(acceptEncoding)
        for _, encoding := range encodings {
            response.EncodeBody(encoding)
        }
        contentEncoding := strings.Join(encodings, ", ")
        if len(contentEncoding) != 0 {
            response.SetHeader("Content-Encoding", contentEncoding)
        }
    }
    if handlerError != nil {
        errorResponse := NewResponse()
        errorResponse.SetStatusCode(500)
        log.Println(fmt.Sprintf("[HTTP/%.1f] %s - %s : %d", protocol, method, path, errorResponse.statusCode))
        _, writeError := connection.Write(errorResponse.Build())
        if writeError != nil {
            log.Fatalln("Failed to write response: ", writeError.Error())
        }
        return true
    } else {
        _, writeError := connection.Write(response.Build())
        log.Println(fmt.Sprintf("[HTTP/%.1f] %s - %s : %d", protocol, method, path, response.statusCode))
        if writeError != nil {
            log.Fatalln("Failed to write response: ", writeError.Error())
        }
        return true
    }
    return false
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

func (server *Server) RegisterHandler(method string, path string, name string, handlerFunc HandlerFunc) {
    found := server.patternRegex.FindAllString(path, -1)
    replacedPath := "^" + path + "$"
    for _, str := range found {
        str = strings.TrimRight(strings.TrimLeft(str, "{"), "}")
        this := fmt.Sprintf("%s%s}", "{", str)
        that := fmt.Sprintf("(?P<%s>\\w+)", str)
        replacedPath = strings.Replace(replacedPath, this, that, 1)
    }
    regex := regexp.MustCompile(replacedPath)
    id := uuid.New()
    server.patterns[id] = NewPattern(regex, method)
    server.handlers[id] = NewHandler(name, handlerFunc)
}

func DurationToTook(dur *time.Duration) string {
    if dur.Nanoseconds() < 1000 {
        return fmt.Sprintf("%d ns", dur.Nanoseconds())
    }
    if dur.Microseconds() < 1000 {
        return fmt.Sprintf("%d µs", dur.Microseconds())
    }
    return fmt.Sprintf("%d ms", dur.Milliseconds())
}

func NewServer(protocol string, address string, port uint32) Server {
    fullAddress := fmt.Sprintf("%s:%d", address, port)

    return Server{
        protocol:    protocol,
        address:     address,
        port:        port,
        fullAddress: fullAddress,
        handlers: map[uuid.UUID]Handler{
            {0}: {
                name: "Default",
                handler: func(_ *Request, res *Response, _ *Context) error {
                    res.SetStatusCode(200)
                    return nil
                },
            },
        },
        patterns: map[uuid.UUID]Pattern{
            {0}: {
                regex:  regexp.MustCompile("^/$"),
                method: "get",
            },
        },
        patternRegex:  regexp.MustCompile("\\{([a-z]\\w+)}"),
        encodingRegex: regexp.MustCompile(".+/.+"),
    }
}
