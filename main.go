package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/maskarb/skarbek-dev/internal/sensor"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// Credentials which stores google ids.
type Credentials struct {
	Cid     string `json:"client_id"`
	Csecret string `json:"client_secret"`
}

type Web struct {
	Creds Credentials `json:"web"`
}

var (
	web_creds Web
	conf      *oauth2.Config
	state     string
	src       = rand.NewSource(time.Now().UnixNano())
)

func RandStringBytesMaskImprSrcSB(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
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
	state = RandStringBytesMaskImprSrcSB(10)
	if _, err := w.Write([]byte("<html><title>Golang Google</title> <body> <a href='" + getLoginURL(state) + "'><button>Login with Google!</button> </a> </body></html>")); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("Hello World")); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
	}
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
		// SetDBMiddleware,
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/sensor", sensor.NewSensorServer().Routes())
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
	if err := json.Unmarshal(file, &web_creds); err != nil {
		log.Fatalf("json unmarshal err: %v", err)
	}

	conf = &oauth2.Config{
		ClientID:     web_creds.Creds.Cid,
		ClientSecret: web_creds.Creds.Csecret,
		RedirectURL:  "https://skarbek.dev/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
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
