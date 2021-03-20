package taskstore

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/maskarb/skarbek-dev/internal/constants"
)

type taskServer struct {
	store *TaskStore
}

func NewTaskServer() *taskServer {
	store := NewTaskStore()
	return &taskServer{store: store}
}

func (ts *taskServer) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", ts.createTaskHandler)
	router.With(paginate).Get("/", ts.getAllTasksHandler)
	router.Delete("/", ts.deleteAllTasksHandler)
	router.Route("/{taskID}", func(r chi.Router) {
		r.Use(ts.taskCtx)                   // Load the *Article on the request context
		r.Get("/", ts.getTaskHandler)       // GET /task/123
		r.Put("/", ts.updateTaskHandler)    // PUT /task/123
		r.Delete("/", ts.deleteTaskHandler) // DELETE /task/123
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

func (ts *taskServer) createTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling task create at %s\n", req.URL.Path)

	// Types used internally in this handler to (de-)serialize the request and
	// response from/to JSON.
	type RequestTask struct {
		Text string    `json:"text"`
		Tags []string  `json:"tags"`
		Due  time.Time `json:"due"`
	}

	type ResponseId struct {
		Id int `json:"id"`
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
	var rt RequestTask
	if err := dec.Decode(&rt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := ts.store.CreateTask(rt.Text, rt.Tags, rt.Due)
	render.JSON(w, req, ResponseId{Id: id})
}

func (ts *taskServer) getAllTasksHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get all tasks at %s\n", req.URL.Path)

	allTasks := ts.store.GetAllTasks()
	render.JSON(w, req, allTasks)
}

func (ts *taskServer) getTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling get task at %s\n", req.URL.Path)

	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the TaskCtx
	// middleware. The worst case, the recoverer middleware will save us.
	task := req.Context().Value(constants.TaskContextID).(Task)
	render.JSON(w, req, task)
}

func (ts *taskServer) taskCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskID, err := strconv.Atoi(chi.URLParam(r, "taskID"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		task, err := ts.store.GetTask(taskID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), constants.TaskContextID, task)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (ts *taskServer) deleteTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling delete task at %s\n", req.URL.Path)

	task := req.Context().Value(constants.TaskContextID).(Task)

	if err := ts.store.DeleteTask(task.Id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

func (ts *taskServer) deleteAllTasksHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling delete all tasks at %s\n", req.URL.Path)
	if err := ts.store.DeleteAllTasks(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ts *taskServer) updateTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("handling update task at %s\n", req.URL.Path)

	task := req.Context().Value(constants.TaskContextID).(Task)

	if err := ts.store.UpdateTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

// func (ts *taskServer) tagHandler(w http.ResponseWriter, req *http.Request) {
// 	log.Printf("handling tasks by tag at %s\n", req.URL.Path)

// 	if req.Method != http.MethodGet {
// 		http.Error(w, fmt.Sprintf("expect method GET /tag/<tag>, got %v", req.Method), http.StatusMethodNotAllowed)
// 		return
// 	}

// 	path := strings.Trim(req.URL.Path, "/")
// 	pathParts := strings.Split(path, "/")
// 	if len(pathParts) < 2 {
// 		http.Error(w, "expect /tag/<tag> path", http.StatusBadRequest)
// 		return
// 	}
// 	tag := pathParts[1]

// 	tasks := ts.store.GetTasksByTag(tag)
// 	render.JSON(w, req, tasks)
// }

// func (ts *taskServer) dueHandler(w http.ResponseWriter, req *http.Request) {
// 	log.Printf("handling tasks by due at %s\n", req.URL.Path)

// 	if req.Method != http.MethodGet {
// 		http.Error(w, fmt.Sprintf("expect method GET /due/<date>, got %v", req.Method), http.StatusMethodNotAllowed)
// 		return
// 	}

// 	path := strings.Trim(req.URL.Path, "/")
// 	pathParts := strings.Split(path, "/")

// 	badRequestError := func() {
// 		http.Error(w, fmt.Sprintf("expect /due/<year>/<month>/<day>, got %v", req.URL.Path), http.StatusBadRequest)
// 	}
// 	if len(pathParts) != 4 {
// 		badRequestError()
// 		return
// 	}

// 	year, err := strconv.Atoi(pathParts[1])
// 	if err != nil {
// 		badRequestError()
// 		return
// 	}
// 	month, err := strconv.Atoi(pathParts[2])
// 	if err != nil || month < int(time.January) || month > int(time.December) {
// 		badRequestError()
// 		return
// 	}
// 	day, err := strconv.Atoi(pathParts[3])
// 	if err != nil {
// 		badRequestError()
// 		return
// 	}

// 	tasks := ts.store.GetTasksByDueDate(year, time.Month(month), day)
// 	render.JSON(w, req, tasks)
// }
