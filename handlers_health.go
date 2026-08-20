package main

import (
	"fmt"
	"net/http"
)

func ReadinessHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)

	// Writes out OK to body of response
	_, err := writer.Write([]byte("OK"))
	if err != nil {
		err := fmt.Errorf("Error in readiness handler body write: %w", err)
		fmt.Println(err.Error())
		return
	}
}
