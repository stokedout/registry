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

package patch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apigee/registry/cmd/registry/core"
	"github.com/apigee/registry/pkg/connection"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/names"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExport(t *testing.T) {
	tests := []struct {
		desc string
		root string
	}{
		{
			desc: "sample",
			root: "testdata/sample-hierarchical",
		},
	}
	for _, test := range tests {
		project := names.Project{ProjectID: "patch-export-test"}
		// Make an admin client and use it to create the project.
		ctx := context.Background()
		adminClient, err := connection.NewAdminClient(ctx)
		if err != nil {
			t.Fatalf("Setup: failed to create client: %+v", err)
		}
		if err = adminClient.DeleteProject(ctx, &rpc.DeleteProjectRequest{
			Name:  project.String(),
			Force: true,
		}); err != nil && status.Code(err) != codes.NotFound {
			t.Fatalf("Setup: failed to delete test project: %s", err)
		}
		if _, err := adminClient.CreateProject(ctx, &rpc.CreateProjectRequest{
			ProjectId: project.ProjectID,
			Project:   &rpc.Project{},
		}); err != nil {
			t.Fatalf("Setup: Failed to create test project: %s", err)
		}
		// Set the configured registry.project to the test project.
		config, err := connection.ActiveConfig()
		if err != nil {
			t.Fatalf("Setup: Failed to get registry configuration: %s", err)
		}
		config.Project = project.ProjectID
		connection.SetConfig(config)
		// Make a registry client and use it to apply the test data.
		registryClient, err := connection.NewRegistryClient(ctx)
		if err != nil {
			t.Fatalf("Setup: Failed to create registry client: %s", err)
		}
		if err := Apply(ctx, registryClient, test.root, project.String()+"/locations/global", true, 1); err != nil {
			t.Fatalf("Apply() returned error: %s", err)
		}

		t.Cleanup(func() {
			if err := adminClient.DeleteProject(ctx, &rpc.DeleteProjectRequest{
				Name:  project.String(),
				Force: true,
			}); err != nil {
				t.Logf("Cleanup: Failed to delete test project: %s", err)
			}
			adminClient.Close()
			registryClient.Close()
		})

		t.Run(test.desc+"-project", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportProject(ctx, registryClient, project, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export project: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-api", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPI(ctx, registryClient, project.Api("registry"), false, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export api: %s", err)
			}
			wait()
			compareExportedFiles(t, test.root, "apis/registry/info.yaml", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-api-recursive", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPI(ctx, registryClient, project.Api("registry"), true, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export api: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "apis/registry", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-version", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPIVersion(ctx, registryClient, project.Api("registry").Version("v1"), false, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export version: %s", err)
			}
			wait()
			compareExportedFiles(t, test.root, "apis/registry/versions/v1/info.yaml", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-version-recursive", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPIVersion(ctx, registryClient, project.Api("registry").Version("v1"), true, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export version: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "apis/registry/versions/v1", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-spec", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPISpec(ctx, registryClient, project.Api("registry").Version("v1").Spec("openapi"), false, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export spec: %s", err)
			}
			wait()
			compareExportedFiles(t, test.root, "apis/registry/versions/v1/specs/openapi/info.yaml", tempDir, project.ProjectID)
			compareExportedFiles(t, test.root, "apis/registry/versions/v1/specs/openapi/openapi.yaml", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-spec-recursive", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPISpec(ctx, registryClient, project.Api("registry").Version("v1").Spec("openapi"), true, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export spec: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "apis/registry/versions/v1/specs/openapi", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-deployment", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPIDeployment(ctx, registryClient, project.Api("registry").Deployment("prod"), false, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export deployment: %s", err)
			}
			wait()
			compareExportedFiles(t, test.root, "apis/registry/deployments/prod/info.yaml", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-deployment-recursive", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportAPIDeployment(ctx, registryClient, project.Api("registry").Deployment("prod"), true, tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export deployment: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "apis/registry/deployments/prod", tempDir, project.ProjectID)
		})

		t.Run(test.desc+"-artifact", func(t *testing.T) {
			tempDir := t.TempDir()
			taskQueue, wait := core.WorkerPool(ctx, 1)
			err = ExportArtifact(ctx, registryClient, project.Api("registry").Artifact("api-references"), tempDir, taskQueue)
			if err != nil {
				t.Errorf("Failed to export artifact: %s", err)
			}
			wait()
			compareExportedDirectories(t, test.root, "apis/registry/artifacts", tempDir, project.ProjectID)
		})
	}
}

func compareExportedDirectories(t *testing.T, root, top, tempDir, projectID string) {
	if err := filepath.Walk(filepath.Join(root, top), func(refFilename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		refBytes, err := os.ReadFile(refFilename)
		if err != nil {
			return err
		}
		newFilename := filepath.Join(tempDir, projectID, strings.TrimPrefix(refFilename, root))
		newBytes, err := os.ReadFile(newFilename)
		if err != nil {
			return err
		}
		if diff := cmp.Diff(newBytes, refBytes); diff != "" {
			return fmt.Errorf("mismatched export %s %+v", newFilename, diff)
		}
		return nil
	}); err != nil {
		t.Errorf("Failed comparison: %s", err)
	}
}

func compareExportedFiles(t *testing.T, root, file, tempDir, projectID string) {
	refFilename := filepath.Join(root, file)
	refBytes, err := os.ReadFile(refFilename)
	if err != nil {
		t.Errorf("Failed to read file %s", refFilename)
	}
	newFilename := filepath.Join(tempDir, projectID, file)
	newBytes, err := os.ReadFile(newFilename)
	if err != nil {
		t.Errorf("Failed to read file %s", newFilename)
	}
	if diff := cmp.Diff(newBytes, refBytes); diff != "" {
		t.Errorf("Mismatched export %s %+v", newFilename, diff)
	}
}
