package main

type Headers = map[string]string

type Params = map[string]string

type HandlerFunc = func(*Request, *Response, *Context) error
