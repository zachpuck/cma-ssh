package apiserver

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Server struct {
	Manager manager.Manager
}
