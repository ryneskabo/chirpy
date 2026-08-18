package main

import ("fmt";
	"net/http";
)

func main() {
	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr: ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Errorf("%w",err)
	}
}
