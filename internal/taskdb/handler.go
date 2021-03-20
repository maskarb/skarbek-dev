package taskdb

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/maskarb/skarbek-dev/internal/constants"
	"github.com/maskarb/skarbek-dev/internal/models"
)

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", createTaskHandler)
	router.With(paginate).Get("/", getAllTasksHandler)
	router.Delete("/", deleteAllTasksHandler)
	router.Route("/{taskID}", func(r chi.Router) {
		r.Use(taskCtx)                   // Load the *Article on the request context
		r.Get("/", getTaskHandler)       // GET /task/123
		r.Delete("/", deleteTaskHandler) // DELETE /task/123
	})
	return router
}

// paginate is a stub, but very possible to implement middleware logic
// to handle the request params for handling a paginated request.
func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// just a stub.. some ideas are to look at URL query params for something like
		// the page number, or the limit, and send a query cursor down the chain
		next.ServeHTTP(w, r)
	})
}

func createTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling task create at %s\n", req.URL.Path)

	type ResponseId struct {
		Id uint `json:"id"`
	}

	// Enforce a JSON Content-Type.
	contentType := req.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if mediatype != "application/json" {
		http.Error(w, "expect application/json Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	var t models.Task
	if err := dec.Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := t.CreateTask(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.JSON(w, req, ResponseId{Id: t.ID})
}

func getAllTasksHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get all tasks at %s\n", req.URL.Path)

	allTasks, err := models.GetAllTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.JSON(w, req, allTasks)
}

func getTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get task at %s\n", req.URL.Path)

	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the TaskCtx
	// middleware. The worst case, the recoverer middleware will save us.
	task := req.Context().Value(constants.TaskContextID).(models.Task)
	render.JSON(w, req, task)
}

func taskCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskID, err := strconv.Atoi(chi.URLParam(r, "taskID"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var t models.Task
		if err := t.GetTask(taskID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), constants.TaskContextID, t)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func deleteTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling delete task at %s\n", req.URL.Path)

	task := req.Context().Value(constants.TaskContextID).(models.Task)

	if err := task.DeleteTask(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteAllTasksHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling delete all tasks at %s\n", req.URL.Path)
	if err := models.DeleteAllTasks(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
