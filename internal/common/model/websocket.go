package model

type Message struct {
	Type  string `json:"type"`
	Token string `json:"token,omitempty"`
	Data  string `json:"data,omitempty"`
}
