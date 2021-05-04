package controller

import (
	"strings"

	"istio.io/api/networking/v1beta1"
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

func createStringMatch(s string) *v1beta1.StringMatch {
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
