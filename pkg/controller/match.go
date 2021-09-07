package controller

import (
	"strings"

	"istio.io/api/networking/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
)

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code adapted from
// https://github.com/istio/istio/blob/985d7c3b444f039c21e1489f40b751fb584d3a15/pilot/pkg/config/kube/ingress/conversion.go#L155
func createStringMatch(path networkingv1.HTTPIngressPath) *v1beta1.StringMatch {
	var stringMatch *v1beta1.StringMatch

	if path.PathType != nil {
		switch *path.PathType {
		case networkingv1.PathTypeExact:
			stringMatch = &v1beta1.StringMatch{
				MatchType: &v1beta1.StringMatch_Exact{Exact: path.Path},
			}
		case networkingv1.PathTypePrefix:
			// From the spec: /foo/bar matches /foo/bar/baz, but does not match /foo/barbaz
			// Envoy prefix match behaves differently, so insert a / if we don't have one
			path := path.Path
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}
			stringMatch = &v1beta1.StringMatch{
				MatchType: &v1beta1.StringMatch_Prefix{Prefix: path},
			}
		default:
			// Fallback to the string matching
			stringMatch = createFallbackStringMatch(path.Path)
		}
	} else {
		stringMatch = createFallbackStringMatch(path.Path)
	}

	return stringMatch
}

// Code taken from:
// https://github.com/istio/istio/blob/985d7c3b444f039c21e1489f40b751fb584d3a15/pilot/pkg/config/kube/ingress/conversion.go#L309
func createFallbackStringMatch(s string) *v1beta1.StringMatch {
	if s == "" {
		return nil
	}

	// Note that this implementation only converts prefix and exact matches, not regexps.

	// Replace e.g. "foo.*" with prefix match
	if strings.HasSuffix(s, ".*") {
		return &v1beta1.StringMatch{
			MatchType: &v1beta1.StringMatch_Prefix{Prefix: strings.TrimSuffix(s, ".*")},
		}
	}
	if strings.HasSuffix(s, "/*") {
		return &v1beta1.StringMatch{
			MatchType: &v1beta1.StringMatch_Prefix{Prefix: strings.TrimSuffix(s, "*")},
		}
	}

	// Replace e.g. "foo" with a exact match
	return &v1beta1.StringMatch{
		MatchType: &v1beta1.StringMatch_Exact{Exact: s},
	}
}
