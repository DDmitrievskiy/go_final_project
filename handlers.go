package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type addTaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type addTaskResponse struct {
	ID    int64  `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req addTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Ошибка десериализации JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		sendError(w, "Не указан заголовок задачи", http.StatusBadRequest)
		return
	}

	now := time.Now()
	nowParsed := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	today := now.Format("20060102")

	var taskDate time.Time
	if req.Date == "" {
		taskDate = now
	} else {
		var err error
		taskDate, err = time.ParseInLocation("20060102", req.Date, time.Local)
		if err != nil {
			sendError(w, "Некорректный формат даты", http.StatusBadRequest)
			return
		}
	}

	finalDate := taskDate.Format("20060102")

	if taskDate.Before(nowParsed) {
		if req.Repeat == "" {
			finalDate = now.Format("20060102")
		} else {
			next, err := NextDate(nowParsed, finalDate, req.Repeat)
			if err != nil {
				sendError(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req.Repeat == "d 1" && next == today {
				finalDate = nowParsed.Format("20060102")
			} else {
				finalDate = next
			}
		}
	}

	if req.Repeat != "" {
		if _, err := NextDate(nowParsed, finalDate, req.Repeat); err != nil {
			sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	res, err := db.Exec(
		`INSERT INTO scheduler (date, title, comment, repeat) 
         VALUES (?, ?, ?, ?)`,
		finalDate, req.Title, req.Comment, req.Repeat,
	)
	if err != nil {
		sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		sendError(w, "Ошибка получения ID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(addTaskResponse{ID: id})
}

func sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(addTaskResponse{Error: message})
}

func NextDateHandler(w http.ResponseWriter, r *http.Request) {

	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	if nowStr == "" || dateStr == "" || repeat == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Invalid 'now' format", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("20060102", dateStr)
	if err != nil {
		http.Error(w, "Invalid 'date' format", http.StatusBadRequest)
		return
	}

	nextDate, err := NextDate(now, date.Format("20060102"), repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(nextDate)); err != nil {
		log.Printf("Response write failed: %v", err)
	}
}

type Task struct {
	ID      int64  `db:"id"`
	Date    string `db:"date"`
	Title   string `db:"title"`
	Comment string `db:"comment"`
	Repeat  string `db:"repeat"`
}

type TasksResponse struct {
	Tasks []TaskJSON `json:"tasks"`
}

type TaskJSON struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	search := r.FormValue("search")
	var tasks []Task
	var query string
	args := []interface{}{}

	if search != "" {
		if date, err := time.Parse("02.01.2006", search); err == nil {
			formattedDate := date.Format("20060102")
			query = `SELECT * FROM scheduler WHERE date = ? ORDER BY date ASC LIMIT 50`
			args = append(args, formattedDate)
		} else {
			searchParam := "%" + strings.ReplaceAll(search, "%", "\\%") + "%"
			query = `SELECT * FROM scheduler 
                     WHERE title LIKE ? OR comment LIKE ? 
                     ORDER BY date ASC LIMIT 50`
			args = append(args, searchParam, searchParam)
		}
	} else {
		query = `SELECT * FROM scheduler ORDER BY date ASC LIMIT 50`
	}

	err := db.Select(&tasks, query, args...)
	if err != nil {
		sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	response := TasksResponse{
		Tasks: make([]TaskJSON, 0),
	}
	for _, task := range tasks {
		response.Tasks = append(response.Tasks, TaskJSON{
			ID:      strconv.FormatInt(task.ID, 10),
			Date:    task.Date,
			Title:   task.Title,
			Comment: task.Comment,
			Repeat:  task.Repeat,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(response)
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		sendError(w, "Не указан идентификатор", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendError(w, "Некорректный идентификатор", http.StatusBadRequest)
		return
	}

	var task Task
	err = db.Get(&task, `
        SELECT id, date, title, comment, repeat 
        FROM scheduler 
        WHERE id = ?`, id)
	if err != nil {
		sendError(w, "Задача не найдена", http.StatusNotFound)
		return
	}
	response := TaskJSON{
		ID:      strconv.FormatInt(task.ID, 10),
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(response)
}

type updateTaskRequest struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Ошибка десериализации JSON", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		sendError(w, "Не указан заголовок задачи", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		sendError(w, "Не указан идентификатор задачи", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		sendError(w, "Некорректный идентификатор задачи", http.StatusBadRequest)
		return
	}
	var exists bool
	err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM scheduler WHERE id = ?)", id)
	if err != nil || !exists {
		sendError(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	now := time.Now()
	nowParsed := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	today := now.Format("20060102")

	var taskDate time.Time
	if req.Date == "" {
		taskDate = nowParsed
	} else {
		taskDate, err = time.ParseInLocation("20060102", req.Date, time.Local)
		if err != nil {
			sendError(w, "Некорректный формат даты", http.StatusBadRequest)
			return
		}
	}

	finalDate := taskDate.Format("20060102")
	if taskDate.Before(nowParsed) {
		if req.Repeat == "" {
			finalDate = nowParsed.Format("20060102")
		} else {
			next, err := NextDate(nowParsed, finalDate, req.Repeat)
			if err != nil {
				sendError(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req.Repeat == "d 1" && next == today {
				finalDate = nowParsed.Format("20060102")
			} else {
				finalDate = next
			}
		}
	}

	if req.Repeat != "" {
		if _, err := NextDate(nowParsed, finalDate, req.Repeat); err != nil {
			sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	_, err = db.Exec(`
        UPDATE scheduler 
        SET date = ?, title = ?, comment = ?, repeat = ?
        WHERE id = ?`,
		finalDate, req.Title, req.Comment, req.Repeat, id,
	)
	if err != nil {
		sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(struct{}{})
}

func markTaskDoneHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	if idStr == "" {
		sendError(w, "Не указан идентификатор задачи", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendError(w, "Некорректный идентификатор задачи", http.StatusBadRequest)
		return
	}

	var task struct {
		Date   string `db:"date"`
		Repeat string `db:"repeat"`
	}
	err = db.Get(&task, "SELECT date, repeat FROM scheduler WHERE id = ?", id)
	if err != nil {
		sendError(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	if task.Repeat == "" {
		_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	} else {
		now := time.Now()
		nextDate, err := NextDate(
			now,
			task.Date,
			task.Repeat,
		)
		if err != nil {
			sendError(w, fmt.Sprintf("Ошибка расчета даты: %v", err), http.StatusBadRequest)
			return
		}

		_, err = db.Exec(
			"UPDATE scheduler SET date = ? WHERE id = ?",
			nextDate,
			id,
		)
		if err != nil {
			sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}
	}

	if err != nil {
		sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(struct{}{})
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.FormValue("id")
	if idStr == "" {
		sendError(w, "Не указан идентификатор задачи", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendError(w, "Некорректный идентификатор задачи", http.StatusBadRequest)
		return
	}

	var taskExists bool
	err = db.Get(&taskExists, "SELECT EXISTS(SELECT 1 FROM scheduler WHERE id = ?)", id)
	if err != nil || !taskExists {
		sendError(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		sendError(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(struct{}{})
}
