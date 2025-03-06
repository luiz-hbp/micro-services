package main

import (
	"fmt"
	"log"
	"net/http"
)

const webPort = "80"

func main() {
	app := NewConfig(webPort)

	log.Printf("Starting broker service on port %s\n", webPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.Routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}

}
