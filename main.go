package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Credentials which stores google ids.
type Credentials struct {
	Cid     string `json:"cid"`
	Csecret string `json:"csecret"`
}

// User is a retrieved and authentiacted user.
type User struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Gender        string `json:"gender"`
}

var (
	cred   Credentials
	conf   *oauth2.Config
	sensor *bsbmp.BMP
	state  string
)

// Environment is the retreived environment properties.
type Environment struct {
	Temperature *float32 `json:"temperature,omitempty"`
	Humidity    *float32 `json:"humidity,omitempty"`
	Pressue     *float32 `json:"pressure,omitempty"`
	Altitude    *float32 `json:"altitude,omitempty"`
}

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

	// Create new connection to i2c-bus on 1 line with address 0x77.
	// Use i2cdetect utility to find device address over the i2c-bus
	i2c, err := i2c.NewI2C(0x77, 1)
	if err != nil {
		log.Fatalf("new_i2c error: %v", err)
	}

	sensor, err = bsbmp.NewBMP(bsbmp.BME280, i2c)
	if err != nil {
		log.Fatalf("new_bmp error: %v", err)
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
		abortWithError(w, r, http.StatusUnauthorized, fmt.Errorf("Invalid session state: %s", state))
		return
	}

	tok, err := conf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)
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
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Write([]byte("Hello World"))
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()

	for k, v := range values {
		fmt.Fprintf(w, "%v (type: %T) => %v\n", k, k, v)
	}
}

func tempHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	env := &Environment{}
	if err := env.getEnvironment(); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}
	json.NewEncoder(w).Encode(env)

}

func (e *Environment) getEnvironment() error {
	// Uncomment next line to supress verbose output
	//logger.ChangePackageLogLevel("i2c", logger.InfoLevel)

	// Uncomment next line to supress verbose output
	//logger.ChangePackageLogLevel("bsbmp", logger.InfoLevel)

	// Read temperature in celsius degree
	t, err := sensor.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return fmt.Errorf("read temperature error: %v", err)
	}
	e.Temperature = &t
	log.Printf("Temprature = %v*C\n", t)
	// Read atmospheric pressure in pascal
	p, err := sensor.ReadPressurePa(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return fmt.Errorf("read pressure (pascal) error: %v", err)
	}
	e.Pressue = &p
	log.Printf("Pressure = %v Pa\n", p)
	// Read relative humidity in %RH
	supported, rh, err := sensor.ReadHumidityRH(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return fmt.Errorf("read humidity (%%rh) error: %v", err)
	}
	e.Humidity = &rh
	if !supported {
		log.Printf("Sensor does not support relative humidity")
		e.Humidity = nil
	}
	log.Printf("Relative Humidity = %v %%RH\n", rh)
	// Read atmospheric altitude in meters above sea level, if we assume
	// that pressure at see level is equal to 101325 Pa.
	a, err := sensor.ReadAltitude(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return fmt.Errorf("read altitude error: %v", err)
	}
	e.Altitude = &a
	log.Printf("Altitude = %v m\n", a)

	return nil
}

func main() {
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("skarbek.dev"),
		Cache:      autocert.DirCache("certs"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/temp", tempHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/auth", authHandler)

	mux.HandleFunc("/greet/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/greet/"):]
		fmt.Fprintf(w, "Hello %s\n", name)
	})

	server := &http.Server{
		Addr:    ":https",
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}
	go http.ListenAndServe(":http", m.HTTPHandler(nil))

	log.Printf("starting server: %s", server.Addr)
	log.Fatal(server.ListenAndServeTLS("", ""))
}
