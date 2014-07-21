package util

import (
	"runtime"
	"errors"
	"log"
)

func hereInfo() (fun string, file string, line int, err error) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "<unknown>", "<unknown>", -1, errors.New("Unable to recover call context")
	}
	fun = runtime.FuncForPC(pc).Name()
	return
}

func WithHere(cb func(string, string, int)) {
	fun, file, line, err := hereInfo()
	if err != nil {
		log.Println(err)
		return
	}
	cb(fun, file, line)
}
