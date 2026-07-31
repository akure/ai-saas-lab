package kernel

import (
	"context"
	"fmt"
)

// Policy is a named yes/no rule check — auth validity, quota, blacklists.
// Registering by name lets any module contribute rules other modules can
// require, without those modules importing each other.
type Policy interface {
	Name() string
	Evaluate(ctx context.Context, subject any) (bool, error)
}

func (a *App) RegisterPolicy(p Policy) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.policies[p.Name()] = p
}

// CheckPolicies runs a named checklist — every policy must pass.
func (a *App) CheckPolicies(ctx context.Context, subject any, names ...string) error {
	for _, name := range names {
		a.mu.RLock()
		p, ok := a.policies[name]
		a.mu.RUnlock()
		if !ok {
			return fmt.Errorf("unknown policy: %q", name)
		}
		allowed, err := p.Evaluate(ctx, subject)
		if err != nil {
			return fmt.Errorf("policy %q errored: %w", name, err)
		}
		if !allowed {
			return fmt.Errorf("policy denied: %q", name)
		}
	}
	return nil
}

// FuncPolicy lets a module register a policy as a plain function instead of
// a full type — handy for small, module-local rules.
type FuncPolicy struct {
	PolicyName string
	Fn         func(ctx context.Context, subject any) (bool, error)
}

func (f FuncPolicy) Name() string { return f.PolicyName }
func (f FuncPolicy) Evaluate(ctx context.Context, subject any) (bool, error) {
	return f.Fn(ctx, subject)
}
