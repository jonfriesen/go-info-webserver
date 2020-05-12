package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	fmt.Println("Starting server on port 8080")
	http.HandleFunc("/", infoServer)
	http.ListenAndServe(":8080", nil)
}

func infoServer(w http.ResponseWriter, r *http.Request) {
	logWebRequest(r)

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprint(w, "error getting hostname: ", err)
		return
	}
	fmt.Fprintf(w, "hostname: %s\n", hostname)

	fmt.Fprintln(w, "\n\nRuntime Environment Variables:")
	for _, e := range os.Environ() {
		fmt.Fprintln(w, e)
	}

	fmt.Fprintln(w, "\n\nBuildtime Environment Variables:")
	b, err := ioutil.ReadFile("./build-time-envs") // just pass the file name
	if err != nil {
		fmt.Print(err)
	}
	fmt.Fprintln(w, string(b))
}

func logWebRequest(r *http.Request) {

	d, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println("failed to dump http request %v", err)
		return
	}

	fmt.Println("Request:\n" + string(d) + "\n")
}
