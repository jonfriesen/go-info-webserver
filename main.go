package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	buildVariables, err := loadBuildVars()
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", infoServer)
	r.HandleFunc("/envs/build/{buildVar}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		buildVar := vars["buildVar"]

		v, ok := buildVariables[buildVar]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, v)
	})
	r.HandleFunc("/envs/run/{runVar}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		runVar := vars["runVar"]

		v := os.Getenv(runVar)
		if v == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, v)
	})

	fmt.Println("Starting server on port 8080")
	http.ListenAndServe(":8080", r)
}

func infoServer(w http.ResponseWriter, r *http.Request) {
	logWebRequest(r)

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprint(w, "error getting hostname: ", err)
		return
	}
	fmt.Fprintf(w, "hostname: %s\n", hostname)
	
	fmt.Fprintf(w, "server time: %s\n", time.Now().String())

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
		fmt.Printf("failed to dump http request %v\n", err)
		return
	}

	fmt.Println("Request:\n" + string(d) + "\n")
}

func loadBuildVars() (map[string]string, error) {
	bindVars := make(map[string]string)

	file, err := os.Open("./build-time-envs")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		v := strings.SplitN(scanner.Text(), "=", 2)
		if len(v) >= 2 {
			bindVars[v[0]] = v[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return bindVars, nil

}
