package kernel

import (
	"net/http"
	"sync"
)

// App is the plug board every module registers into. It knows nothing about
// AI completions, billing, or auth specifically — it only knows how to hold
// named things and hand them back out. All real behavior lives in modules.
type App struct {
	mu sync.RWMutex

	Config *Config
	Events *EventBus
	Store  *Store
	Mux    *http.ServeMux // shared HTTP router every HTTP-facing module registers routes into

	encoders map[string]Encoder
	messages map[string]MessageDescriptor
	handlers map[string]MessageHandler
	policies map[string]Policy
	modules  []Module
}

func NewApp(cfg *Config) *App {
	return &App{
		Config:   cfg,
		Events:   NewEventBus(),
		Store:    NewStore(),
		Mux:      http.NewServeMux(),
		encoders: make(map[string]Encoder),
		messages: make(map[string]MessageDescriptor),
		handlers: make(map[string]MessageHandler),
		policies: make(map[string]Policy),
	}
}
