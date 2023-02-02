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
	r.HandleFunc("/upload", uploadFile)

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

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		fmt.Println("get method hit, return html")
		fmt.Fprint(w, getUploadForm())
	}
	// guard non-post methods
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "Method %s not supported.", r.Method)
		return
	}

	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("", "upload-*.png")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	_, err = tempFile.Write(fileBytes)
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	// return that we have successfully uploaded our file!
	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

// i am lazy
func getUploadForm() string {
	return `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Document</title>
  </head>
  <body>
    <form
      enctype="multipart/form-data"
      action="/upload"
      method="POST"
    >
      <input type="file" name="myFile" multiple/>
      <input type="submit" value="upload" />
    </form>
  </body>
</html>
	`
}
