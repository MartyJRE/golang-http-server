package main

import (
    "bytes"
    "compress/gzip"
    "fmt"
    "log"
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

func (response *Response) EncodeBody(encoding string) {
    switch encoding {
    case GZip:
        {
            buff := bytes.Buffer{}
            gz := gzip.NewWriter(&buff)
            defer func(gz *gzip.Writer) {
                err := gz.Close()
                if err != nil {
                    log.Fatalln("Failed to close gzip writer!", err)
                }
            }(gz)

            _, err := gz.Write(response.body)
            if err != nil {
                log.Fatalln("Failed to write to gzip writer!", err)
            }
            err = gz.Flush()
            if err != nil {
                log.Fatalln("Failed to flush gzip writer!", err)
            }
            response.SetBody(buff.Bytes())
        }
    }
}
