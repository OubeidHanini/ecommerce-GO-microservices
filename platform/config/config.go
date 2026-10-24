// Package config reads service settings from the environment.
//
// It collects every problem instead of stopping at the first one, so a misconfigured
// service lists all missing settings at once and refuses to start, rather than failing
// later with an obscure error.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Env struct {
	errs []error
}

func New() *Env {
	return &Env{}
}

func (e *Env) Get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (e *Env) Required(key string) string {
	v := os.Getenv(key)
	if v == "" {
		e.errs = append(e.errs, fmt.Errorf("%s is required", key))
	}
	return v
}

// Secret reads the file named by KEY_FILE, falling back to KEY.
// Files are how Docker and Kubernetes mount secrets: unlike environment variables,
// they don't show up in `docker inspect`, in /proc/<pid>/environ or in child processes.
func (e *Env) Secret(key string) string {
	if path := os.Getenv(key + "_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			e.errs = append(e.errs, fmt.Errorf("%s_FILE: %w", key, err))
			return ""
		}
		v := strings.TrimRight(string(b), "\r\n")
		if v == "" {
			e.errs = append(e.errs, fmt.Errorf("%s_FILE: %s is empty", key, path))
		}
		return v
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	e.errs = append(e.errs, fmt.Errorf("%s_FILE or %s is required", key, key))
	return ""
}

// Err returns every problem found so far, or nil.
func (e *Env) Err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration: %w", errors.Join(e.errs...))
}
