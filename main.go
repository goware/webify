package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

var (
	flags = flag.NewFlagSet("webify", flag.ExitOnError)
	port  = flags.String("port", "3000", "http server port")
	host  = flags.String("host", "0.0.0.0", "http server hostname")
	dir   = flags.String("dir", ".", "directory to serve")
	cache = flags.Bool("cache", false, "enable Cache-Control for content")
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

	if *cache {
		r.Use(CacheControl)
	} else {
		r.Use(middleware.NoCache)
	}

	cors := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(cors.Handler)

	workDir, _ := os.Getwd()
	FileServer(r, "/", http.Dir(workDir))

	// Serve it up!
	err := http.ListenAndServe(addr, r)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Head(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

func CacheControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=31536000")
		h.ServeHTTP(w, r)
	})
}
