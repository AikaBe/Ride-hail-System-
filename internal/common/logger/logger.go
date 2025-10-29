package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Структура лога
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Hostname  string `json:"hostname"`
	RequestID string `json:"request_id"`
	RideID    string `json:"ride_id,omitempty"`
	Error     *struct {
		Msg   string `json:"msg"`
		Stack string `json:"stack"`
	} `json:"error,omitempty"`
}

// hostname сервиса
var hostname, _ = os.Hostname()

// Имя сервиса (можно установить при старте)
var serviceName = "unknown-service"

// Установить имя сервиса
func SetServiceName(name string) {
	serviceName = name
}

// INFO лог
func Info(action, message, requestID, rideID string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "INFO",
		Service:   serviceName,
		Action:    action,
		Message:   message,
		Hostname:  hostname,
		RequestID: requestID,
		RideID:    rideID,
	}
	output(entry)
}

// DEBUG лог
func Debug(action, message, requestID, rideID string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "DEBUG",
		Service:   serviceName,
		Action:    action,
		Message:   message,
		Hostname:  hostname,
		RequestID: requestID,
		RideID:    rideID,
	}
	output(entry)
}

// WARN лог
func Warn(action, message, requestID, rideID, errMsg string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "WARN",
		Service:   serviceName,
		Action:    action,
		Message:   message,
		Hostname:  hostname,
		RequestID: requestID,
		RideID:    rideID,
		Error: &struct {
			Msg   string `json:"msg"`
			Stack string `json:"stack"`
		}{
			Msg:   errMsg,
			Stack: "",
		},
	}
	output(entry)
}

// ERROR лог
func Error(action, message, requestID, rideID, errStack string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "ERROR",
		Service:   serviceName,
		Action:    action,
		Message:   message,
		Hostname:  hostname,
		RequestID: requestID,
		RideID:    rideID,
		Error: &struct {
			Msg   string `json:"msg"`
			Stack string `json:"stack"`
		}{
			Stack: errStack,
		},
	}
	output(entry)
}

// Вспомогательная функция для вывода JSON в stdout
func output(entry LogEntry) {
	jsonData, _ := json.Marshal(entry)
	fmt.Println(string(jsonData))
}
