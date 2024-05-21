package main

type Handler struct {
    handler        HandlerFunc
    wrappedHandler HandlerFunc
    name           string
    middlewares    []MiddlewareFunc
}

func NewHandler(name string, handlerFunc HandlerFunc) Handler {
    return Handler{
        name:        name,
        handler:     handlerFunc,
        middlewares: make([]MiddlewareFunc, 0),
    }
}

func (handler *Handler) RegisterMiddleware(middleware MiddlewareFunc) {
    handler.middlewares = append(handler.middlewares, middleware)
}

func (handler *Handler) GetHandler() func(*Request, *Response, *Context) error {
    if handler.wrappedHandler != nil {
        return handler.wrappedHandler
    }
    // TODO: do the MWs
    return handler.wrappedHandler
}
