package utils

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

type ErrorData struct {
	StatusCode int
	Message    string
}

func RenderErrorPage(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	cwd, _ := os.Getwd()
	var templates = template.Must(template.ParseFiles(cwd + "/error.html"))
	data := ErrorData{
		StatusCode: statusCode,
		Message:    message,
	}
	err := templates.ExecuteTemplate(w, cwd+"/error.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
