// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package coreintegration

import (
	"fmt"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/palantir/pkg/bearertoken"
)

// NewUserClient builds a go-oikumenea SDK client that forwards the given bearer token unchanged —
// the caller's own Google ID token, extracted from the incoming request's Authorization header by
// the transport layer. Never widens what the caller can do (D-Facade: OpenFaithMap makes zero
// authorization decisions of its own); go-oikumenea's PDP decides every call for real, same as
// NewServiceClient's service-principal path but for a real person instead of the machine subject.
func NewUserClient(baseURL, token string, insecureSkipVerify bool) (*oikumenea.Client, error) {
	opts := []oikumenea.Option{}
	if insecureSkipVerify {
		opts = append(opts, oikumenea.WithInsecureSkipVerify())
	}
	c, err := oikumenea.New(baseURL, bearertoken.Token(token), opts...)
	if err != nil {
		return nil, fmt.Errorf("coreintegration: dial go-oikumenea as user: %w", err)
	}
	return c, nil
}
