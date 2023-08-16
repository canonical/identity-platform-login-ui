package status

import (
	"context"
)

type ServiceInterface interface {
	KratosStatus(context.Context) bool
	HydraStatus(context.Context) bool
}
