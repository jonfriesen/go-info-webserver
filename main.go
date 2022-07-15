package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/xo/dburl"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	buildVariables, err := loadBuildVars()
	if err != nil {
		log.Println(err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", infoServer)
	r.HandleFunc("/mongo", testMongoConnection)
	r.HandleFunc("/mysql", testMYSQLConnection)
	r.HandleFunc("/postgres", testPostgresConnection)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	bindAddr := fmt.Sprintf(":%s", port)

	fmt.Printf("==> Server listening at %s 🚀\n", bindAddr)

	err = http.ListenAndServe(bindAddr, r)
	if err != nil {
		panic(err)
	}
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

func testMongoConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logWebRequest(r)

	ca := os.Getenv("CA_CERT")
	if ca == "" {
		fmt.Fprintln(w, "CA_CERT env var missing")
		return
	}

	mongoURL := os.Getenv("DATABASE_URL")
	if mongoURL == "" {
		fmt.Fprintln(w, "DATABASE_URL connection string missing")
		return
	}

	opts := options.Client()
	opts.ApplyURI(mongoURL)

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(ca))
	if !ok {
		fmt.Fprintln(w, "appending certs from pem")
		return
	}
	opts.SetTLSConfig(&tls.Config{
		RootCAs: roots,
	})

	client, err := mongo.NewClient(opts)
	if err != nil {
		fmt.Fprintln(w, "client creation failed")
		fmt.Fprintln(w, err.Error())
		return
	}
	err = client.Connect(ctx)
	if err != nil {
		fmt.Fprintln(w, "connection failed")
		fmt.Fprintln(w, err.Error())
		return
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Fprintln(w, "ping failed")
		fmt.Fprintln(w, err.Error())
		return
	}

	fmt.Fprintln(w, "connection & ping succesful")
}

func testMYSQLConnection(w http.ResponseWriter, r *http.Request) {
	logWebRequest(r)

	uri := os.Getenv("DATABASE_URL")
	if uri == "" {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "no DATABASE_URL env var")
		return
	}

	dbURL, err := dburl.Parse(uri)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "parsing DATABASE_URL")
		return
	}

	dbPassword, _ := dbURL.User.Password()
	dbName := strings.Trim(dbURL.Path, "/")
	connectionString := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8&parseTime=true",
		dbURL.User.Username(), dbPassword, dbURL.Hostname(), dbURL.Port(), dbName)

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error connecting to the database: "+err.Error())
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error connecting to the database: "+err.Error())
		return
	}

	fmt.Fprint(w, "Successfully connected and pinged mysql.")
}

func testPostgresConnection(w http.ResponseWriter, r *http.Request) {
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "no DATABASE_URL env var")
		return
	}

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error connecting to the database: "+err.Error())
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error connecting to the database: "+err.Error())
		return
	}

	fmt.Fprint(w, "Successfully connected and pinged postgres.")
}
