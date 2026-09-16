// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"net/url"
	"slices"
	"strings"
)

const resourceParam = "resource"

// resourceIndicators splits the RFC 8707 "resource" query parameters of the
// authorization URL into valid indicators (absolute URI, no fragment) and the rest.
func resourceIndicators(requestURL string) (valid, invalid []string) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, nil
	}

	for _, resource := range u.Query()[resourceParam] {
		if isResourceIndicator(resource) {
			valid = append(valid, resource)
		} else {
			invalid = append(invalid, resource)
		}
	}

	return valid, invalid
}

// isResourceIndicator reports whether value is an absolute URI without a fragment.
func isResourceIndicator(value string) bool {
	if strings.ContainsAny(value, "# \t\r\n\\") {
		return false
	}

	u, err := url.Parse(value)

	return err == nil && u.IsAbs()
}

// permittedResources splits resources into those the registered audience permits
// and those it rejects; an empty registered audience permits nothing.
func permittedResources(registered, resources []string) (permitted, rejected []string) {
	for _, resource := range resources {
		if slices.ContainsFunc(registered, func(audience string) bool { return audiencePermits(audience, resource) }) {
			permitted = append(permitted, resource)
		} else {
			rejected = append(rejected, resource)
		}
	}

	return permitted, rejected
}

// audiencePermits mirrors fosite.DefaultAudienceMatchingStrategy: equal scheme, host
// and opaque part, and a resource path equal to or nested under the registered path.
func audiencePermits(registered, resource string) bool {
	ru, err := url.Parse(registered)
	if err != nil {
		return false
	}

	u, err := url.Parse(resource)
	if err != nil {
		return false
	}

	if u.Scheme != ru.Scheme || u.Host != ru.Host || u.Opaque != ru.Opaque {
		return false
	}

	allowed := strings.TrimRight(ru.Path, "/")

	return u.Path == ru.Path || u.Path == allowed || strings.HasPrefix(u.Path, allowed+"/")
}

// mergeAudience appends the resources missing from audience into a new slice.
func mergeAudience(audience, resources []string) []string {
	if len(resources) == 0 {
		return audience
	}

	merged := make([]string, 0, len(audience)+len(resources))
	merged = append(merged, audience...)

	for _, resource := range resources {
		if !slices.Contains(merged, resource) {
			merged = append(merged, resource)
		}
	}

	return merged
}
