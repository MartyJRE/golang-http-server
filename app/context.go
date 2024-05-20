package main

import (
	"regexp"
)

type Context struct {
	params  Params
	pattern *regexp.Regexp
	path    string
}

func ContextFromRequest(request *Request) (Context, error) {
	path, pathErr := request.Path()
	if pathErr != nil {
		return Context{}, pathErr
	}
	return Context{
		path: path,
	}, nil
}

func (context *Context) PopulateParams(pattern *regexp.Regexp) {
	matches := pattern.FindStringSubmatch(context.path)
	for nameIdx, name := range pattern.SubexpNames() {
		if name == "" {
			continue
		}
		context.params[name] = matches[nameIdx]
	}
}
