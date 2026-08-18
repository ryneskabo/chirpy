package main

import ("fmt";
	"net/http";
)

func ReadinessHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)

	// Writes out OK to body of response
	_, err := writer.Write([]byte("OK"))
	if err != nil {
		err := fmt.Errorf("Error in readiness handler body write: %w", err)
		fmt.Println(err.Error())
	}
}

func main() { 
	const port = "8080"

	serveMux := http.NewServeMux()
	strippedPathHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", strippedPathHandler)

	// Handles Readiness Endpoint, shows if server is up and running
	serveMux.HandleFunc("/healthz", ReadinessHandler)

	// Initializes server struct and starts it using the serveMux handler
	server := http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}
	err := server.ListenAndServe()
	if err != nil {
		err := fmt.Errorf("Error in listen and serve %w: ",err)
		fmt.Println(err.Error())
	}
}
