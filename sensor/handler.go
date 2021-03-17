package sensor

import (
	"fmt"
	"log"
	"net/http"

	"github.com/d2r2/go-bsbmp"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func abortWithError(w http.ResponseWriter, r *http.Request, status int, err error) {
	w.WriteHeader(status)
	if status >= 400 {
		fmt.Fprintf(w, "error: %v\n", err)
	}
}

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/{sensorID}", GetSensor)
	router.Get("/", GetAllSensors)
	return router
}

func GetSensor(w http.ResponseWriter, r *http.Request) {
	sensorID := chi.URLParam(r, "sensorID")
	env := &Environment{}
	if err := env.getEnvironment(); err != nil {
		abortWithError(w, r, http.StatusBadRequest, err)
		return
	}

	if fmt.Sprintf("%d", *env.SensorID) != sensorID {
		log.Printf("Sensor ID = %v | %v\n", sensorID, fmt.Sprintf("%d", *env.SensorID))
		http.NotFound(w, r)
		return
	}

	render.JSON(w, r, env)
}

func GetAllSensors(w http.ResponseWriter, r *http.Request) {
	sensors := []bsbmp.BMP{*piSensor}
	render.JSON(w, r, sensors)
}
