package main

import (
    "fmt"
)

type Response struct {
    protocol   float32
    statusCode uint16
    message    string
    headers    Headers
    body       []byte
}

func NewResponse() Response {
    return Response{protocol: 1.1, headers: make(Headers)}
}

func (response *Response) SetHeader(key string, value string) {
    response.headers[key] = value
}

var statusToMessage = map[uint16]string{
    200: "OK",
    201: "Created",
    404: "Not Found",
    500: "Internal Server Error",
}

func (response *Response) SetStatusCode(statusCode uint16) {
    response.statusCode = statusCode
    if message, ok := statusToMessage[response.statusCode]; ok {
        response.message = message
    } else {
        response.message = ""
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
