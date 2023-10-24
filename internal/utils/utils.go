package utils

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Result[T any] struct {
	Val T
	Err error
}

// simply serialize to json and write struct as http response with status code
func WriteJsonAndStatusInRespone[T any](w http.ResponseWriter, j *T, status int) {
	w.WriteHeader(status)
	jsn, _ := json.Marshal(*j)
	w.Write(jsn)
}

// simply serialize to json and write struct as http response with status code
func WriteStringAndStatusInRespone(w http.ResponseWriter, j *string, status int) {
	w.WriteHeader(status)
	w.Write([]byte(*j))
}

func GetAddrFromStr(addrNPort *string) string {
	return strings.Split(
		*addrNPort, ":",
	)[0]
}
