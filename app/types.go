package main

type Headers = map[string]string

type Params = map[string]string

type Handler = func(*Request, *Response, *Context) error
