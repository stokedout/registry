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

// Api recommended_version must be an ApiVersion that is a child of this Api.
package rule102

import (
	"context"
	"fmt"
	"strings"

	"github.com/apigee/registry/cmd/registry/cmd/check/lint"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/names"
)

var ruleNum = 102
var ruleName = lint.NewRuleName(ruleNum, "api-recommended-deployment-ref")

// AddRules accepts a register function and registers each of
// this rules' checks to the RuleRegistry.
func AddRules(r lint.RuleRegistry) error {
	return r.Register(
		ruleNum,
		recommendedDeploymentRef,
	)
}

var recommendedDeploymentRef = &lint.ApiRule{
	Name: ruleName,
	OnlyIf: func(a *rpc.Api) bool {
		return strings.TrimSpace(a.RecommendedDeployment) != ""
	},
	ApplyToApi: func(ctx context.Context, a *rpc.Api) []lint.Problem {
		deploymentName, err := names.ParseDeployment(a.RecommendedDeployment)
		if err != nil {
			return []lint.Problem{{
				Severity:   lint.ERROR,
				Message:    fmt.Sprintf(`recommended_deployment %q is not a valid ApiDeployment name.`, a.RecommendedDeployment),
				Suggestion: fmt.Sprintf(`Parse error: %s`, err),
			}}
		}

		apiName, _ := names.ParseApi(a.Name) // name assumed to be valid
		if deploymentName.Api() != apiName {
			return []lint.Problem{{
				Severity:   lint.ERROR,
				Message:    fmt.Sprintf(`recommended_deployment %q is not a child of this Api.`, a.RecommendedDeployment),
				Suggestion: `Correct the recommended_deployment.`,
			}}
		}

		registryClient := lint.RegistryClient(ctx)
		if _, err := registryClient.GetApiDeployment(ctx, &rpc.GetApiDeploymentRequest{
			Name: a.RecommendedDeployment,
		}); err != nil {
			return []lint.Problem{{
				Severity:   lint.ERROR,
				Message:    fmt.Sprintf(`recommended_deployment %q not found in registry.`, a.RecommendedDeployment),
				Suggestion: `Correct the recommended_deployment.`,
			}}
		}

		return nil
	},
}
