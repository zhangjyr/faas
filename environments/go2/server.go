package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/valyala/fasthttp"
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

var userFunc fasthttp.RequestHandler

func loadPlugin(codePath, entrypoint string) fasthttp.RequestHandler {

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
	case *fasthttp.RequestHandler:
		return *h
	case func(*fasthttp.RequestCtx):
		return h
	default:
		panic("Entry point not found: bad type")
	}
}

func specializeHandlerV2(ctx *fasthttp.RequestCtx) {
	if userFunc != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Write([]byte("Not a generic container"))
		return
	}

	var loadreq FunctionLoadRequest
	err := json.Unmarshal(ctx.PostBody(), &loadreq)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
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
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.Write([]byte(path + ": not found"))
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

func readinessProbeHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
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

	m := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/":
			// Check x-function header, ignore if not match
			// log.Printf("Incoming funciton request: %s\n", r.Header.Get("X-FUNCTION"))
			if string(ctx.Request.Header.Peek("X-FUNCTION")) != functionName {
				log.Printf("Ignore unexpected funciton request, %s expected\n", functionName)
				ctx.Hijack(func(c net.Conn) {
					c.Close()
				})
				return
			}

			if userFunc == nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.Write([]byte("Generic container: no requests supported"))
				return
			}
			userFunc(ctx)
		case "/healthz":
			readinessProbeHandler(ctx)
		case "/v2/specialize":
			specializeHandlerV2(ctx)
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	fmt.Printf("Listening on %d ...\n", port)
	serverStopped := make(chan struct{})
	go func() {
		if err := fasthttp.ListenAndServe(fmt.Sprintf(":%d", port), m); err != nil {
			fmt.Printf("Error ListenAndServe: %v\n", err)
			close(serverStopped)
		}
	}()

	if _, _, err := fasthttp.Get(nil, fmt.Sprintf("http://localhost:8079/_/ready/%d", port)); err != nil {
		fmt.Printf("Error on register environment: %v\n", err)
	}

	<-serverStopped
}
