// Copyright 2023 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rule108

import (
	"context"
	"testing"

	"github.com/apigee/registry/cmd/registry/cmd/check/lint"
	"github.com/apigee/registry/rpc"
	"github.com/google/go-cmp/cmp"
)

func TestAddRules(t *testing.T) {
	if err := AddRules(lint.NewRuleRegistry()); err != nil {
		t.Errorf("AddRules got an error: %v", err)
	}
}

func Test_externalChannelURIFormat(t *testing.T) {
	prob := []lint.Problem{{
		Severity:   lint.ERROR,
		Message:    `external_channel_uri must be an absolute URI.`,
		Suggestion: `Ensure external_channel_uri includes a host.`,
	}}

	for _, tt := range []struct {
		in       string
		expected []lint.Problem
	}{
		{"", nil},
		{"x", prob},
		{"https://google.com/ok#yes?ok=true", nil},
	} {
		t.Run(tt.in, func(t *testing.T) {
			a := &rpc.ApiDeployment{
				EndpointUri: tt.in,
			}
			if externalChannelUriFormat.OnlyIf(a) {
				got := externalChannelUriFormat.ApplyToApiDeployment(context.Background(), a)
				if diff := cmp.Diff(got, tt.expected); diff != "" {
					t.Errorf("unexpected diff: (-want +got):\n%s", diff)
				}
			}
		})
	}
}
