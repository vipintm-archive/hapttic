package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf8"
)

const version = "1.0.0"

// This is a subset of http.Request with the types changed so that we can marshall it.
type marshallableRequest struct {
	Method string
	URL    string
	Proto  string
	Host   string

	Header http.Header

	ContentLength int64
	Body          string
	Form          url.Values
	PostForm      url.Values
}

func init() {
	log.SetOutput(os.Stdout)
}

// handleFuncWithScriptFileName constructs our handleFunc
func handleFuncWithScriptFileName(scriptFileName string) func(s http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// First we make sure the script we are about to execute still exists
		if _, err := os.Stat(scriptFileName); os.IsNotExist(err) {
			log.Fatal("The request handling script " + scriptFileName + " does not exist.")
		}

		// This parses the request body
		bodyBuffer := new(bytes.Buffer)
		bodyBuffer.ReadFrom(req.Body)
		body := bodyBuffer.String()

		req.ParseForm()

		// Copy over all the information from the request we are interested in
		marshallableReq := marshallableRequest{
			Method: req.Method,
			URL:    req.URL.String(),
			Proto:  req.Proto,
			Host:   req.Host,

			Header: req.Header,

			ContentLength: req.ContentLength,
			Body:          body,
			Form:          req.Form,
			PostForm:      req.PostForm,
		}

		// Try to convert to JSON. This shouldn't fail
		requestJSON, err := json.Marshal(marshallableReq)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Executing " + scriptFileName)

		// Execute the request handling script
		out, err := exec.Command("/bin/bash", scriptFileName, string(requestJSON)).Output()

		if err != nil {
			// If there was an error, we return a response with status code 500
			res.WriteHeader(http.StatusInternalServerError)
			io.WriteString(res, "500 Internal Server Error")
		} else {
			// Otherwise we return the output of our script
			res.Write(out)
		}
	}
}

func main() {
	// Parse command line args
	printVersion := flag.Bool("v", false, "Print version and exit.")
	printUsage := flag.Bool("u", false, "Print usage and exit")
	host := flag.String("h", "", "The host to bind to, e.g. 0.0.0.0 or localhost.")
	port := flag.String("p", "8080", "The port to listen on.")
	userScriptFileName := flag.String("f", "./hapttic_request_handler.sh", "The script that is called to handle requests.")
	flag.Parse()

	if *printVersion {
		fmt.Fprintf(os.Stderr, version+"\n")
		os.Exit(0)
	}

	if *printUsage {
		fmt.Fprintf(os.Stderr, "Usage of hapttic:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if utf8.RuneCountInString(*userScriptFileName) == 0 {
		log.Fatal("The path to the request handling script can not be empty.")
	}

	scriptFileName, err := filepath.Abs(*userScriptFileName)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handleFuncWithScriptFileName(scriptFileName))

	addr := *host + ":" + *port
	log.Println("Thanks for using hapttic")
	log.Println(fmt.Sprintf("Listening on %s", addr))
	log.Println(fmt.Sprintf("Forwarding requests to %s", scriptFileName))
	log.Fatal(http.ListenAndServe(addr, nil))
}
