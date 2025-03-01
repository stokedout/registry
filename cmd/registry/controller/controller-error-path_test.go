// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"testing"

	"github.com/apigee/registry/pkg/connection"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/test/seeder"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func deleteProject(
	ctx context.Context,
	client connection.AdminClient,
	t *testing.T,
	projectID string) {
	t.Helper()
	req := &rpc.DeleteProjectRequest{
		Name:  "projects/" + projectID,
		Force: true,
	}
	err := client.DeleteProject(ctx, req)
	if err != nil && status.Code(err) != codes.NotFound {
		t.Fatalf("Failed DeleteProject(%v): %s", req, err.Error())
	}
}

// Tests for error paths in the controller

func TestControllerErrors(t *testing.T) {
	tests := []struct {
		desc              string
		generatedResource *rpc.GeneratedResource
	}{
		{
			desc: "Non-existing reference in dependencies",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/artifacts/lintstats-gnostic",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.spec", // Correct pattern should be: $resource.version
					},
				},
				Action: "registry compute lintstats $resource.spec --linter gnostic",
			},
		},
		{
			desc: "Incorrect reference keyword",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/specs/-/artifacts/lint-gnostic",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.apispec", // Correct pattern should be: $resource.spec
					},
				},
				Action: "registry compute lint $resource.apispec --linter gnostic",
			},
		},
		{
			desc: "Nonexistent dependency resource",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/artifacts/lintstats-gnostic",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.version/artifacts/lint-gnostic", // There is no version level lint-gnostic artifact in the registry
					},
				},
				//Correct action should be "registry compute lintstats $resource.version --linter gnostic"
				Action: "registry compute lintstats $resource.version/artifacts/lint-gnostic --linter gnostic",
			},
		},
		{
			desc: "Incorrect reference in action",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/specs/-/artifacts/lintstats-gnostic",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.spec",
					},
				},
				Action: "registry compute lintstats $resource.artifact --linter gnostic", // Correct reference should be: $resource.spec
			},
		},
		{
			desc: "Incorrect resource pattern",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/specs/-/artifacts/lintstats-gnostic", // Correct pattern should be: apis/-/versions/-/specs/-/artifacts/lintstats-gnostic
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.spec",
					},
				},
				Action: "registry compute lintstats $resource.specs --linter gnostic",
			},
		},
		{
			desc: "Incorrect matching",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/artifacts/summary",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.api/versions/-/artifacts/complexity", // Correct pattern should be: $resource.version/artifacts/vocabulary
					},
					{
						Pattern: "$resource.version/artifacts/vocabulary",
					},
				},
				Action: "registry compute summary $resource.version",
			},
		},
		{
			desc: "Incorrect action reference",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/versions/-/artifacts/score",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.version/-/artifacts/complexity",
					},
				},
				// Correct action should be: "compute summary $resource.version/artifacts/complexity"
				Action: "registry compute summary $resource.api/versions/-/artifacts/complexity",
			},
		},
		{
			desc: "Missing reference",
			generatedResource: &rpc.GeneratedResource{
				Pattern: "apis/-/artifacts/summary",
				Dependencies: []*rpc.Dependency{
					{
						Pattern: "$resource.api/versions/-/artifacts/complexity",
					},
					{
						Pattern: "$resource.api/versions/-/artifacts/vocabulary",
					},
				},
				Action: "registry compute summary $resource", // Correct action should be: compute summary $resource.api
			},
		},
	}

	const projectID = "controller-error-demo"
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			registryClient, err := connection.NewRegistryClient(ctx)
			if err != nil {
				t.Fatalf("Failed to create client: %+v", err)
			}
			t.Cleanup(func() { registryClient.Close() })

			adminClient, err := connection.NewAdminClient(ctx)
			if err != nil {
				t.Fatalf("Failed to create client: %+v", err)
			}
			t.Cleanup(func() { adminClient.Close() })

			// Setup
			deleteProject(ctx, adminClient, t, "controller-test")
			t.Cleanup(func() { deleteProject(ctx, adminClient, t, "controller-test") })

			client := seeder.Client{
				RegistryClient: registryClient,
				AdminClient:    adminClient,
			}

			seed := []seeder.RegistryResource{
				&rpc.ApiSpec{
					Name:     "projects/controller-test/locations/global/apis/petstore/versions/1.0.0/specs/openapi.yaml",
					MimeType: gzipOpenAPIv3,
				},
				&rpc.ApiSpec{
					Name:     "projects/controller-test/locations/global/apis/petstore/versions/1.0.1/specs/openapi.yaml",
					MimeType: gzipOpenAPIv3,
				},
				&rpc.ApiSpec{
					Name:     "projects/controller-test/locations/global/apis/petstore/versions/1.1.0/specs/openapi.yaml",
					MimeType: gzipOpenAPIv3,
				},
			}

			if err := seeder.SeedRegistry(ctx, client, seed...); err != nil {
				t.Fatalf("Setup: failed to seed registry: %s", err)
			}

			lister := &RegistryLister{RegistryClient: registryClient}

			// Test GeneratedResource pattern
			actions, err := processManifestResource(ctx, lister, projectID, test.generatedResource)
			if err == nil {
				t.Errorf("Expected processManifestResource() to return an error, got: %v", actions)
			}
		})
	}
}
