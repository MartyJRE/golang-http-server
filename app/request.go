package main

import (
    "errors"
    "strconv"
    "strings"
)

type Request struct {
    buffer      []byte
    splitLines  []string
    requestLine []string
    path        string
    method      string
    protocol    float32
    headers     Headers
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

func (parser *Request) Headers() (Headers, error) {
    if parser.headers != nil {
        return parser.headers, nil
    }
    splitLines := parser.SplitLines()

    headers := splitLines[1 : len(splitLines)-2]
    parser.headers = make(Headers)
    for _, header := range headers {
        parts := strings.SplitN(header, string(':'), 2)
        if len(parts) != 2 {
            return nil, errors.New("header was malformed")
        }
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        parser.headers[key] = val
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

func (parser *Request) GetHeader(header string) (string, error) {
    headers, err := parser.Headers()
    if err != nil {
        return "", err
    }
    return headers[header], nil
}
