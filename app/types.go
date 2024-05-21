package main

type Headers = map[string]string

type Params = map[string]string

type HandlerFunc = func(*Request, *Response, *Context) error

type MiddlewareFunc = func(*Request, *Response, *Context, HandlerFunc) error
