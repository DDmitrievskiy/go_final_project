package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

func main() {
	var err error
	db, err = InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	port := getPort()

	r := mux.NewRouter()

	r.HandleFunc("/api/nextdate", NextDateHandler).Methods("GET")
	r.HandleFunc("/api/tasks", getTasksHandler).Methods("GET")
	r.HandleFunc("/api/task/done", markTaskDoneHandler).Methods("POST")
	r.HandleFunc("/api/task", addTaskHandler).Methods("POST")
	r.HandleFunc("/api/task", getTaskHandler).Methods("GET")
	r.HandleFunc("/api/task", updateTaskHandler).Methods("PUT")
	r.HandleFunc("/api/task", deleteTaskHandler).Methods("DELETE")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web")))

	log.Printf("Сервер запущен на порту %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}

func getPort() int {
	if envPort := os.Getenv("TODO_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			return p
		}
	}

	defaultPort := 7540

	return defaultPort
}
