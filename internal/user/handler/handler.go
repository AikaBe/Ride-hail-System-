package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"ride-hail/internal/common/logger"
	"ride-hail/internal/user/handler/dto"
	"ride-hail/internal/user/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	action := "register_user"
	requestID := r.Header.Get("X-Request-ID")

	if r.Method != http.MethodPost {
		logger.Warn(action, "invalid HTTP method", requestID, "", "only POST allowed")
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(action, "failed to decode request body", requestID, "", err.Error())
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info(action, "register request received", requestID, "")

	createdUser, err := h.authService.Register(context.Background(), req)
	if err != nil {
		logger.Error(action, "failed to register user", requestID, "", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Info(action, "user successfully registered", requestID, string(createdUser.ID))

	resp := dto.RegisterResponse{
		UserID: string(createdUser.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	action := "login_user"
	requestID := r.Header.Get("X-Request-ID")

	if r.Method != http.MethodPost {
		logger.Warn(action, "invalid HTTP method", requestID, "", "only POST allowed")
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(action, "failed to decode request body", requestID, "", err.Error())
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info(action, "login request received", requestID, "")

	access, refresh, err := h.authService.Login(context.Background(), req.Email, req.Password)
	if err != nil {
		logger.Error(action, "login failed", requestID, "", err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	logger.Info(action, "user successfully logged in", requestID, "")

	resp := dto.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	action := "refresh_token"
	requestID := r.Header.Get("X-Request-ID")

	if r.Method != http.MethodPost {
		logger.Warn(action, "invalid HTTP method", requestID, "", "only POST allowed")
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(action, "failed to decode request body", requestID, "", err.Error())
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info(action, "refresh token request received", requestID, "")

	resp, err := h.authService.RefreshToken(context.Background(), req)
	if err != nil {
		logger.Error(action, "token refresh failed", requestID, "", err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	logger.Info(action, "token successfully refreshed", requestID, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
