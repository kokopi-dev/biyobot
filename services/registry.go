package services

import (
	"biyobot/configs"
	"encoding/json"
	"fmt"
)

type Registry struct {
	services map[string]configs.Runner
}

func NewRegistry() *Registry {
	return &Registry{services: make(map[string]configs.Runner)}
}

func (r *Registry) Register(name string, svc configs.Runner) {
	r.services[name] = svc
}

func (r *Registry) Run(name string, input json.RawMessage) configs.ServiceResult {
	svc, ok := r.services[name]
	if !ok {
		return configs.Failure(fmt.Sprintf("unknown service: %q", name))
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
