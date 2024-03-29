package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog"
)

var (
	flags = flag.NewFlagSet("webify", flag.ExitOnError)
	port  = flags.String("port", "3000", "http server port")
	host  = flags.String("host", "0.0.0.0", "http server hostname")
	dir   = flags.String("dir", ".", "directory to serve")
	cache = flags.Bool("cache", false, "enable Cache-Control for content")
	debug = flags.Bool("debug", false, "Debug mode, printing all network request details")
	echo  = flags.Bool("echo", false, "Echo back request body, useful for debugging")
)

func main() {
	flags.Parse(os.Args[1:])

	// Setup params
	addr := fmt.Sprintf("%s:%s", *host, *port)
	cwd, _ := os.Getwd()
	if *dir == "" || *dir == "." {
		*dir = cwd
	} else {
		if (*dir)[0:1] != "/" {
			*dir = filepath.Join(cwd, *dir)
		}
		if _, err := os.Stat(*dir); os.IsNotExist(err) {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
	}

	// Print banner
	fmt.Printf("================================================================================\n")
	fmt.Printf("Serving:  %s\n", *dir)
	fmt.Printf("URL:      http://%s\n", addr)
	if *cache {
		fmt.Printf("Cache:    on\n")
	} else {
		fmt.Printf("Cache:    off\n")
	}
	fmt.Printf("================================================================================\n")
	fmt.Printf("\n")

	logger := httplog.NewLogger("", httplog.Options{
		JSON:    false,
		Concise: false,
	})

	// Setup http router with file server
	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(logger))
	if *debug {
		r.Use(DebugLogger)
	}

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

	if *echo {
		r.HandleFunc("/*", echoHandler)
	} else {
		FileServer(r, "/", http.Dir(*dir))
	}

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

func DebugLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		print := func(s string, v []string) {
			fmt.Printf("=> %s: %v\n", s, v)
		}

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		requestURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

		fmt.Println("*** Debug, request headers:*")

		print("URL", []string{requestURL})
		print("Method", []string{r.Method})
		print("Path", []string{r.URL.Path})
		print("RemoteIP", []string{r.RemoteAddr})
		print("Proto", []string{r.Proto})

		for header, values := range r.Header {
			print(header, values)
		}

		if *echo {
			reqBody, _ := ioutil.ReadAll(r.Body)
			print("Body", []string{string(reqBody)})
		}

		fmt.Println("***")

		next.ServeHTTP(w, r)
	})
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// body, _ := ioutil.ReadAll(r.Body)
	w.WriteHeader(200)
	w.Write([]byte(""))
}
