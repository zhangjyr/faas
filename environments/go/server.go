package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/fission/fission/environments/go/context"
)

const (
	CODE_PATH = "./example"
)

var (
	functionName string
)

type (
	FunctionLoadRequest struct {
		// FilePath is an absolute filesystem path to the
		// function. What exactly is stored here is
		// env-specific. Optional.
		FilePath string `json:"filepath"`

		// FunctionName has an environment-specific meaning;
		// usually, it defines a function within a module
		// containing multiple functions. Optional; default is
		// environment-specific.
		FunctionName string `json:"functionName"`

		// URL to expose this function at. Optional; defaults
		// to "/".
		URL string `json:"url"`
	}
)

var userFunc http.HandlerFunc

func loadPlugin(codePath, entrypoint string) http.HandlerFunc {

	// if codepath's a directory, load the file inside it
	info, err := os.Stat(codePath)
	if err != nil {
		panic(err)
	}
	if info.IsDir() {
		files, err := ioutil.ReadDir(codePath)
		if err != nil {
			panic(err)
		}
		if len(files) == 0 {
			panic("No files to load")
		}
		fi := files[0]
		codePath = filepath.Join(codePath, fi.Name())
	}

	fmt.Printf("loading plugin from %v\n", codePath)
	p, err := plugin.Open(codePath)
	if err != nil {
		panic(err)
	}
	sym, err := p.Lookup(entrypoint)
	if err != nil {
		panic("Entry point not found")
	}

	switch h := sym.(type) {
	case *http.Handler:
		return (*h).ServeHTTP
	case *http.HandlerFunc:
		return *h
	case func(http.ResponseWriter, *http.Request):
		return h
	case func(context.Context, http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request) {
			c := context.New()
			h(c, w, r)
		}
	default:
		panic("Entry point not found: bad type")
	}
}

// func specializeHandler(w http.ResponseWriter, r *http.Request) {
// 	if userFunc != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		w.Write([]byte("Not a generic container"))
// 		return
// 	}
//
// 	_, err := os.Stat(CODE_PATH)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			w.WriteHeader(http.StatusNotFound)
// 			w.Write([]byte(CODE_PATH + ": not found"))
// 			return
// 		} else {
// 			panic(err)
// 		}
// 	}
//
// 	fmt.Println("Specializing ...")
// 	userFunc = loadPlugin(CODE_PATH, "Handler")
// 	fmt.Println("Done")
// }

func specializeHandlerV2(w http.ResponseWriter, r *http.Request) {
	if userFunc != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Not a generic container"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var loadreq FunctionLoadRequest
	err = json.Unmarshal(body, &loadreq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	segments := strings.SplitN(loadreq.FunctionName, ".", 2)
	entity := "Handler"
	if len(segments) > 1 {
		entity = segments[1]
	}

	path := filepath.Join(loadreq.FilePath, segments[0])
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(path + ": not found"))
			return
		} else {
			panic(err)
		}
	}

	fmt.Println("Specializing ...")
	userFunc = loadPlugin(path, entity)
	functionName = loadreq.FunctionName
	fmt.Println("Done")
}

func readinessProbeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	var port int
	var specialize string
	flag.IntVar(&port, "port", 0, "Specify the port to listen")
	flag.StringVar(&specialize, "specialize", "", "Specify the module to specialize on starting")
	flag.Parse()

	if port == 0 {
		flag.CommandLine.Usage()
	}

	if len(specialize) > 0 {
		go func() {
			segments := strings.SplitN(specialize, ".", 2)
			entity := "Handler"
			if len(segments) > 1 {
				entity = segments[1]
			}
			userFunc = loadPlugin(filepath.Join(CODE_PATH, segments[0]), entity)
			functionName = specialize
		}()
	}

	http.HandleFunc("/healthz", readinessProbeHandler)
	// http.HandleFunc("/specialize", specializeHandler)
	http.HandleFunc("/v2/specialize", specializeHandlerV2)

	// Generic route -- all http requests go to the user function.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check x-function header, ignore if not match
		// log.Printf("Incoming funciton request: %s\n", r.Header.Get("X-FUNCTION"))
		if r.Header.Get("X-FUNCTION") != functionName {
			log.Printf("Ignore unexpected funciton request, %s expected\n", functionName)
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}

		if userFunc == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Generic container: no requests supported"))
			return
		}
		userFunc(w, r)
	})

	fmt.Printf("Listening on %d ...\n", port)
	serverStopped := make(chan struct{})
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != http.ErrServerClosed {
			fmt.Printf("Error ListenAndServe: %v\n", err)
			close(serverStopped)
		}
	}()

	if _, err := http.Get(fmt.Sprintf("http://localhost:8079/_/ready/%d", port)); err != nil {
		fmt.Printf("Error on register environment: %v\n", err)
	}

	<-serverStopped
}
