package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Tenant represents a tenant entry returned by the external tenants API.
type Tenant struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// Service fetches tenant data from the external tenants API.
type Service struct {
	tenantsAPIURL string
	httpClient    *http.Client
}

func (s *Service) GetUserTenants(ctx context.Context, userID string) ([]Tenant, error) {
	url := fmt.Sprintf("%s/api/v0/users/%s/tenants", s.tenantsAPIURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer a")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenants: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenants API returned status %d", resp.StatusCode)
	}

	var wrapper struct {
		Tenants []Tenant `json:"tenants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode tenants response: %w", err)
	}

	enabled := make([]Tenant, 0, len(wrapper.Tenants))
	for _, t := range wrapper.Tenants {
		if t.Enabled {
			enabled = append(enabled, t)
		}
	}
	return enabled, nil
}

func NewService(tenantsAPIURL string) *Service {
	return &Service{
		tenantsAPIURL: tenantsAPIURL,
		httpClient:    &http.Client{},
	}
}
