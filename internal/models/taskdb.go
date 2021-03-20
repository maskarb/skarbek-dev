package models

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model

	Text string    `json:"text"`
	Tags []Tag     `json:"tags" gorm:"foreignKey:ID"`
	Due  time.Time `json:"due"`
}

type Tag struct {
	ID   uint `gorm:"primaryKey"`
	Text string
}

// CreateTask creates a new task in the store.
func (t *Task) CreateTask() error {
	result := DB.Create(t)
	return result.Error
}

// GetTask retrieves a task from the store, by id. If no such id exists, an
// error is returned.
func (t *Task) GetTask(id int) error {
	result := DB.First(t, id)
	return result.Error
}

// DeleteTask deletes the task with the given id. If no such id exists, an error
// is returned.
func (t *Task) DeleteTask() error {
	result := DB.Delete(t)
	return result.Error
}

// DeleteAllTasks deletes all tasks in the store.
func DeleteAllTasks() error {
	result := DB.Delete(&Task{})
	return result.Error
}

// GetAllTasks returns all the tasks in the store, in arbitrary order.
func GetAllTasks() ([]Task, error) {
	tasks := []Task{}
	result := DB.Find(&tasks)
	return tasks, result.Error
}
