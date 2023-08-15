package status

import (
	"context"
)

type ServiceInterface interface {
	CheckKratosReady(context.Context) (bool, error)
	CheckHydraReady(context.Context) (bool, error)
}
