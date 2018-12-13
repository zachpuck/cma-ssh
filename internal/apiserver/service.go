package apiserver

import (
	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger logr.Logger
)

type Server struct{}

func SetLogger() {
	logger = logf.ZapLogger(false).WithName("HTTP")
}