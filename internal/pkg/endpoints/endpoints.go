// Package endpoints ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package endpoints

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

// TargetRetriever is implemented by any type that can return the URL of a set of Prometheus metrics providers
type TargetRetriever interface {
	GetTargets() ([]Target, error)
	Watch() error
	Name() string
}

// Object represents a kubernetes object like a pod or a service.
type Object struct {
	Name   string
	Kind   string
	Labels labels.Set
}

// Target is a prometheus endpoint which is exposed by an Object.
type Target struct {
	Name      string
	Object    Object
	URL       url.URL
	metadata  labels.Set
	TLSConfig TLSConfig
}

// Metadata returns the Target's metadata, if the current metadata is nil,
// it's constructed from the Target's attributes, saved and returned.
// Subsequent calls will returned the already saved value.
func (t *Target) Metadata() labels.Set {
	if t.metadata == nil {
		metadata := labels.Set{}
		if t.URL.String() != "" {
			metadata["scrapedTargetURL"] = t.URL.String()
		}
		if t.Object.Name != "" {
			metadata["scrapedTargetName"] = t.Object.Name
			metadata["scrapedTargetKind"] = t.Object.Kind
		}
		labels.Accumulate(metadata, t.Object.Labels)

		t.metadata = metadata
	}
	return t.metadata
}

// New returns a Target from the discovered information
func New(name string, addr url.URL, object Object) Target {
	return Target{
		Name:   name,
		Object: object,
		URL:    addr,
	}
}

// EndpointToTarget returns a list of Targets from the provided TargetConfig struct.
// The URL processing for every Target follows the next conventions:
// - if no schema is provided, it assumes http
// - if no path is provided, it assumes /metrics
// For example, hostname:8080 will be interpreted as http://hostname:8080/metrics
func EndpointToTarget(tc TargetConfig) ([]Target, error) {
	targets := make([]Target, 0, len(tc.URLs))
	for _, URL := range tc.URLs {
		t, err := urlToTarget(URL, tc.TLSConfig)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func urlToTarget(URL string, TLSConfig TLSConfig) (Target, error) {
	if !strings.Contains(URL, "://") {
		URL = fmt.Sprint("http://", URL)
	}

	u, err := url.Parse(URL)
	if err != nil {
		return Target{}, err
	}
	if u.Path == "" {
		u.Path = "/metrics"
	}

	return Target{
		Name: u.Host,
		Object: Object{
			Name:   u.Host,
			Kind:   "user_provided",
			Labels: make(labels.Set),
		},
		TLSConfig: TLSConfig,
		URL:       *u,
	}, nil
}
