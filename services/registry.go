package services

import (
	"biyobot/models"
	"encoding/json"
	"fmt"
)

type Registry struct {
	services map[string]models.Runner
}

func NewRegistry() *Registry {
	return &Registry{services: make(map[string]models.Runner)}
}

func (r *Registry) Register(name string, svc models.Runner) {
	r.services[name] = svc
}

func (r *Registry) Run(name string, input json.RawMessage) models.ServiceResult {
	svc, ok := r.services[name]
	if !ok {
		return models.Failure(fmt.Sprintf("unknown service: %q", name))
	}
	return svc.Run(input)
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.services))
	for k := range r.services {
		names = append(names, k)
	}
	return names
}
