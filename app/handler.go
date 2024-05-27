package main

type Handler struct {
    handler     HandlerFunc
    name        string
    middlewares []HandlerFunc
}

func NewHandler(name string, handlerFunc HandlerFunc) Handler {
    return Handler{
        name:        name,
        handler:     handlerFunc,
        middlewares: make([]HandlerFunc, 0),
    }
}
