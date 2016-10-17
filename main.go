package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
)

var (
	flags = flag.NewFlagSet("webify", flag.ExitOnError)
	port  = flags.String("port", "3000", "http server port")
	host  = flags.String("host", "0.0.0.0", "http server hostname")
	dir   = flags.String("dir", ".", "directory to serve")
)

func main() {
	flags.Parse(os.Args[1:])

	// Setup params
	addr := fmt.Sprintf("%s:%s", *host, *port)
	if *dir == "" || *dir == "." {
		cwd, _ := os.Getwd()
		*dir = cwd
	}

	// Print banner
	fmt.Printf("================================================================================\n")
	fmt.Printf("Serving:  %s\n", *dir)
	fmt.Printf("URL:      http://%s\n", addr)
	fmt.Printf("================================================================================\n")
	fmt.Printf("\n")

	// Setup http router with file server
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.FileServer("/", http.Dir(*dir))

	// Serve it up!
	err := http.ListenAndServe(addr, r)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}
