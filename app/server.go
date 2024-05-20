package main

import (
    "errors"
    "fmt"
    "log"
    "net"
    "regexp"
    "strconv"
    "strings"
)

const (
    BufferSize = 1024
)

type Response struct {
    protocol   float32
    statusCode uint16
    message    string
    headers    map[string]string
    body       []byte
}

func NewResponse() Response {
    return Response{protocol: 1.1, headers: make(map[string]string)}
}

func (response *Response) SetHeader(key string, value string) {
    response.headers[key] = value
}

func (response *Response) SetStatusCode(statusCode uint16) {
    response.statusCode = statusCode
    switch response.statusCode {
    case 200:
        {
            response.message = "OK"
        }
    case 404:
        {
            response.message = "Not Found"
        }
    }
}

func (response *Response) Build() []byte {
    return []byte(fmt.Sprintf("HTTP/%.1f %d %s\r\n%s\r\n%s\r\n", response.protocol, response.statusCode, response.message, response.BuildHeaders(), response.body))
}

func (response *Response) BuildHeaders() string {
    result := ""
    for header, value := range response.headers {
        result += fmt.Sprintf("%s: %s\r\n", header, value)
    }
    return result
}

func (response *Response) SetBody(data []byte) {
    contentLength := len(data)
    response.SetHeader("Content-Length", fmt.Sprintf("%d", contentLength))
    response.body = data
}

type Request struct {
    buffer      []byte
    splitLines  []string
    requestLine []string
    path        string
    method      string
    protocol    float32
    headers     map[string]string
    body        []byte
}

func NewRequest(buffer []byte) Request {
    return Request{
        buffer: buffer,
    }
}

func (parser *Request) SplitLines() []string {
    if parser.splitLines == nil {
        parser.splitLines = strings.Split(string(parser.buffer), "\r\n")
    }
    return parser.splitLines
}

func (parser *Request) RequestLine() ([]string, error) {
    splitLines := parser.SplitLines()
    if len(splitLines) < 1 {
        return nil, errors.New("the request was malformed")
    }
    if parser.requestLine == nil {
        parser.requestLine = strings.Split(splitLines[0], string(' '))
    }
    return parser.requestLine, nil
}

func (parser *Request) Path() (string, error) {
    if parser.path != "" {
        return parser.path, nil
    }
    requestLine, err := parser.RequestLine()
    if err != nil {
        return "", err
    }
    if len(requestLine) < 2 {
        return "", errors.New("path not found")
    }
    parser.path = requestLine[1]
    return parser.path, nil
}

func (parser *Request) Method() (string, error) {
    if parser.method != "" {
        return parser.method, nil
    }
    requestLine, err := parser.RequestLine()
    if err != nil {
        return "", err
    }
    if len(requestLine) < 1 {
        return "", errors.New("method not found")
    }
    parser.method = requestLine[0]
    return parser.method, nil
}

func (parser *Request) Protocol() (float32, error) {
    if parser.protocol != 0 {
        return parser.protocol, nil
    }
    requestLine, err := parser.RequestLine()
    if err != nil {
        return 0, err
    }
    if len(requestLine) < 3 {
        return 0, errors.New("protocol not found")
    }
    protocolString := strings.TrimPrefix(requestLine[2], "HTTP/")
    protocol, err := strconv.ParseFloat(protocolString, 32)
    if err != nil {
        return 0, err
    }
    parser.protocol = float32(protocol)
    return parser.protocol, nil
}

func (parser *Request) Headers() (map[string]string, error) {
    if parser.headers != nil {
        return parser.headers, nil
    }
    splitLines := parser.SplitLines()

    headers := splitLines[1 : len(splitLines)-2]
    parser.headers = make(map[string]string)
    for _, header := range headers {
        parts := strings.SplitN(header, string(':'), 2)
        if len(parts) != 2 {
            return nil, errors.New("header was malformed")
        }
        strings.TrimSpace(parts[0])
        strings.TrimSpace(parts[1])
        parser.headers[parts[0]] = parts[1]
    }

    return parser.headers, nil
}

func (parser *Request) Body() []byte {
    if parser.body != nil {
        return parser.body
    }

    splitLines := parser.SplitLines()
    return []byte(splitLines[len(splitLines)-1])
}

type Server struct {
    protocol    string
    address     string
    port        uint32
    fullAddress string
    listener    net.Listener
    handlers    map[uint32]func(*Request, *Response, *Context) error
    patterns    map[uint32]*regexp.Regexp
    maxId       uint32
    regex       *regexp.Regexp
}

func (server *Server) Start() {
    listener, bindError := net.Listen(server.protocol, server.fullAddress)
    if bindError != nil {
        log.Fatalf("Failed to bind to port %d\n", server.port)
    }
    server.listener = listener
    connection, acceptError := server.listener.Accept()
    if acceptError != nil {
        log.Fatalln("Error accepting connection: ", acceptError.Error())
    }
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
            context := Context{params: make(map[string]string)}
            matches := pattern.FindStringSubmatch(path)
            for nameIdx, name := range pattern.SubexpNames() {
                if name == "" {
                    continue
                }
                context.params[name] = matches[nameIdx]
            }
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
    _, bindError = connection.Write(response.Build())
    if bindError != nil {
        log.Fatalln("Failed to write response: ", bindError.Error())
    }
}

type Context struct {
    params map[string]string
}

func (server *Server) RegisterHandler(path string, handler func(*Request, *Response, *Context) error) {
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

func NewServer(protocol string, address string, port uint32) Server {
    fullAddress := fmt.Sprintf("%s:%d", address, port)

    return Server{
        protocol:    protocol,
        address:     address,
        port:        port,
        fullAddress: fullAddress,
        handlers: map[uint32]func(*Request, *Response, *Context) error{
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

func main() {
    server := NewServer("tcp", "0.0.0.0", 4221)
    server.RegisterHandler("/echo/{str}", func(request *Request, response *Response, context *Context) error {
        response.SetStatusCode(200)
        response.SetHeader("Content-Type", "text/plain")
        param := context.params["str"]
        response.SetBody([]byte(param))
        return nil
    })
    server.Start()
}
