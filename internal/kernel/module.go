package kernel

import (
	"context"
	"fmt"
)

// Module is a self-contained feature. Init is where it registers encoders,
// messages, handlers, policies, and HTTP routes into the shared App. Start
// and Stop manage its own background lifecycle (servers, workers).
type Module interface {
	Name() string
	Init(app *App) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func (a *App) RegisterModule(m Module) {
	a.modules = append(a.modules, m)
}

func (a *App) InitAll() error {
	for _, m := range a.modules {
		if err := m.Init(a); err != nil {
			return fmt.Errorf("module %q init failed: %w", m.Name(), err)
		}
	}
	return nil
}

func (a *App) StartAll(ctx context.Context) error {
	for _, m := range a.modules {
		if err := m.Start(ctx); err != nil {
			return fmt.Errorf("module %q start failed: %w", m.Name(), err)
		}
	}
	return nil
}

// StopAll shuts modules down in reverse order — last started, first stopped.
func (a *App) StopAll(ctx context.Context) {
	for i := len(a.modules) - 1; i >= 0; i-- {
		_ = a.modules[i].Stop(ctx)
	}
}
