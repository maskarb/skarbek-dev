package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/maskarb/skarbek-dev/internal/constants"
	"github.com/maskarb/skarbek-dev/internal/models"
	"github.com/maskarb/skarbek-dev/internal/sensor"
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
	rand.Read(b) //nolint
	return base64.StdEncoding.EncodeToString(b)
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
	if _, err := w.Write([]byte("Hello authenticated user: " + string(data))); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state = randToken()
	if _, err := w.Write([]byte("<html><title>Golang Google</title> <body> <a href='" + getLoginURL(state) + "'><button>Login with Google!</button> </a> </body></html>")); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("Hello World")); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
	}
}

// func userHandler(w http.ResponseWriter, r *http.Request) {
// 	values := r.URL.Query()

// 	for k, v := range values {
// 		fmt.Fprintf(w, "%v (type: %T) => %v\n", k, k, v)
// 	}
// }

func SetDBMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), constants.DBContextID, models.DB.WithContext(context.TODO()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		SetDBMiddleware,
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/sensor", sensor.NewSensorServer().Routes())
		// r.Mount("/tasks", taskstore.NewTaskServer().Routes())
		// r.Mount("/tasksdb", taskdb.Routes())
	})
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/auth", authHandler)
	router.HandleFunc("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, %s!\n", chi.URLParam(r, "name"))
	})

	return router
}

func init() {
	file, err := os.ReadFile("/etc/oauth/creds.json")
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	if err := json.Unmarshal(file, &cred); err != nil {
		log.Fatalf("json unmarshal err: %v", err)
	}

	conf = &oauth2.Config{
		ClientID:     cred.Cid,
		ClientSecret: cred.Csecret,
		RedirectURL:  "https://skarbek.dev/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
		Endpoint: google.Endpoint,
	}

	// models.Setup()
}

func main() {

	router := Routes()

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Printf("starting server: %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
