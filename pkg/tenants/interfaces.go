package tenants

import "context"

type ServiceInterface interface {
	GetUserTenants(ctx context.Context, userID string) ([]Tenant, error)
}
