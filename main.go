package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/maskarb/skarbek-dev/sensor"
)

// Credentials which stores google ids.
type Credentials struct {
	Cid     string `json:"client_id"`
	Csecret string `json:"client_secret"`
}

var (
	cred  Credentials
	conf  *oauth2.Config
	state string
)

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func init() {
	file, err := ioutil.ReadFile("./creds.json")
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	json.Unmarshal(file, &cred)

	conf = &oauth2.Config{
		ClientID:     cred.Cid,
		ClientSecret: cred.Csecret,
		RedirectURL:  "https://skarbek.dev/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
		Endpoint: google.Endpoint,
	}
}

func getLoginURL(state string) string {
	return conf.AuthCodeURL(state)
}

func abortWithError(w http.ResponseWriter, r *http.Request, status int, err error) {
	w.WriteHeader(status)
	if status >= 400 {
		fmt.Fprintf(w, "error: %v\n", err)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	// Handle the exchange code to initiate a transport.
	if state != r.FormValue("state") {
		abortWithError(w, r, http.StatusUnauthorized, fmt.Errorf("invalid session state: %s", state))
		return
	}

	tok, err := conf.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}

	client := conf.Client(context.Background(), tok)
	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}
	defer email.Body.Close()
	data, _ := ioutil.ReadAll(email.Body)
	log.Println("Email body: ", string(data))
	w.Write([]byte("Hello authenticated user: " + string(data)))

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state = randToken()
	w.Write([]byte("<html><title>Golang Google</title> <body> <a href='" + getLoginURL(state) + "'><button>Login with Google!</button> </a> </body></html>"))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()

	for k, v := range values {
		fmt.Fprintf(w, "%v (type: %T) => %v\n", k, k, v)
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	targetURL := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
	log.Printf("redirect to: %s", targetURL.String())
	http.Redirect(w, r, targetURL.String(), http.StatusPermanentRedirect)
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON), // Set content-Type headers as application/json
		middleware.Logger,
		middleware.RedirectSlashes,
		middleware.RequestID,
		middleware.Recoverer,
		middleware.Timeout(60*time.Second),
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/sensor", sensor.Routes())
	})
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/auth", authHandler)
	router.HandleFunc("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, %s!\n", chi.URLParam(r, "name"))
	})

	return router
}

func main() {

	router := Routes()

	server := &http.Server{
		Addr:    ":8443",
		Handler: router,
	}
	go http.ListenAndServe(":8080", http.HandlerFunc(redirect))

	log.Printf("starting server: %s", server.Addr)
	log.Fatal(server.ListenAndServeTLS("/etc/letsencrypt/live/skarbek.dev/fullchain.pem", "/etc/letsencrypt/live/skarbek.dev/privkey.pem"))
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	lp := filepath.Join("web", "templates", "layout.html")
	fp := filepath.Join("web", "templates", filepath.Clean(r.URL.Path))

	tmpl, _ := template.ParseFiles(lp, fp)
	tmpl.ExecuteTemplate(w, "layout", nil)
}
