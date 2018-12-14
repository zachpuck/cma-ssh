package apiserver

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger logr.Logger
)

type Server struct {
	Manager manager.Manager
}

func SetLogger() {
	logger = logf.ZapLogger(false).WithName("HTTP")
}
