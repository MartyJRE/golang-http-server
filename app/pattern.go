package main

import (
    "regexp"
)

type Pattern struct {
    regex  *regexp.Regexp
    method string
}

func NewPattern(regex *regexp.Regexp, method string) Pattern {
    return Pattern{
        regex:  regex,
        method: method,
    }
}
