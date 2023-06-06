package rpcapi

import (
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
)

type dependenciesServer struct {
	terraform1.UnimplementedDependenciesServer
}
