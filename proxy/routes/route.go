package routes

import (
	"net/http"
)

func InitMux() {
	router := map[string]func(http.ResponseWriter, *http.Request){
		"/": Process,
	}

	for key, value := range router {
		http.HandleFunc(key, value)
	}
}
