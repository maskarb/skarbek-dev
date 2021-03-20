package sensor

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/maskarb/skarbek-dev/internal/constants"
)

type sensorServer struct {
	store *SensorStore
}

func NewSensorServer() *sensorServer {
	store := NewSensorStore()
	return &sensorServer{store: store}
}

func (ss *sensorServer) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", ss.getAllSensorsHandler)
	router.Route("/{sensorID}", func(r chi.Router) {
		r.Use(ss.sensorCtx)
		r.Get("/", ss.getSensorHandler)
	})
	return router
}

func (ss *sensorServer) sensorCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sensorID, err := strconv.Atoi(chi.URLParam(r, "sensorID"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sensor, err := ss.store.GetSensor(sensorID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), constants.SensorContextID, *sensor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (ss *sensorServer) getSensorHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get task at %s\n", req.URL.Path)

	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the TaskCtx
	// middleware. The worst case, the recoverer middleware will save us.
	sensor := req.Context().Value(constants.SensorContextID).(Sensor)
	render.JSON(w, req, sensor)
}

func (ss *sensorServer) getAllSensorsHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get all sensors at %s\n", req.URL.Path)

	allSensors, err := ss.store.GetAllSensors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.JSON(w, req, allSensors)
}
