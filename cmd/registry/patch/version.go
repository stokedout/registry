// Copyright 2022 Google LLC. All Rights Reserved.
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

	"github.com/apigee/registry/cmd/registry/core"
	"github.com/apigee/registry/gapic"
	"github.com/apigee/registry/pkg/connection"
	"github.com/apigee/registry/pkg/models"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/names"
	"gopkg.in/yaml.v3"
)

// NewApiVersion allows an API version to be individually exported as a YAML file.
func NewApiVersion(ctx context.Context, client *gapic.RegistryClient, message *rpc.ApiVersion, nested bool) (*models.ApiVersion, error) {
	versionName, err := names.ParseVersion(message.Name)
	if err != nil {
		return nil, err
	}

	var specs []*models.ApiSpec
	var artifacts []*models.Artifact
	if nested {
		specs = make([]*models.ApiSpec, 0)
		if err = core.ListSpecs(ctx, client, versionName.Spec("-"), "", false, func(message *rpc.ApiSpec) error {
			spec, err := NewApiSpec(ctx, client, message, true)
			if err != nil {
				return err
			}
			// unset these because they can be inferred
			spec.ApiVersion = ""
			spec.Kind = ""
			spec.Metadata.Parent = ""
			specs = append(specs, spec)
			return nil
		}); err != nil {
			return nil, err
		}
		artifacts, err = collectChildArtifacts(ctx, client, versionName.Artifact("-"))
		if err != nil {
			return nil, err
		}
	}
	return &models.ApiVersion{
		Header: models.Header{
			ApiVersion: RegistryV1,
			Kind:       "Version",
			Metadata: models.Metadata{
				Name:        versionName.VersionID,
				Parent:      names.ExportableName(versionName.Parent(), versionName.ProjectID),
				Labels:      message.Labels,
				Annotations: message.Annotations,
			},
		},
		Data: models.ApiVersionData{
			DisplayName: message.DisplayName,
			Description: message.Description,
			State:       message.State,
			PrimarySpec: message.PrimarySpec,
			ApiSpecs:    specs,
			Artifacts:   artifacts,
		},
	}, nil
}

func applyApiVersionPatchBytes(
	ctx context.Context,
	client connection.RegistryClient,
	bytes []byte,
	parent string) error {
	var version models.ApiVersion
	err := yaml.Unmarshal(bytes, &version)
	if err != nil {
		return err
	}
	return applyApiVersionPatch(ctx, client, &version, parent)
}

func versionName(parent string, metadata models.Metadata) (names.Version, error) {
	if metadata.Parent != "" {
		parent = parent + "/" + metadata.Parent
	}
	api, err := names.ParseApi(parent)
	if err != nil {
		return names.Version{}, err
	}
	return api.Version(metadata.Name), nil
}

func applyApiVersionPatch(
	ctx context.Context,
	client connection.RegistryClient,
	version *models.ApiVersion,
	parent string) error {
	name, err := versionName(parent, version.Metadata)
	if err != nil {
		return err
	}
	req := &rpc.UpdateApiVersionRequest{
		ApiVersion: &rpc.ApiVersion{
			Name:        name.String(),
			DisplayName: version.Data.DisplayName,
			Description: version.Data.Description,
			State:       version.Data.State,
			PrimarySpec: version.Data.PrimarySpec,
			Labels:      version.Metadata.Labels,
			Annotations: version.Metadata.Annotations,
		},
		AllowMissing: true,
	}
	_, err = client.UpdateApiVersion(ctx, req)
	if err != nil {
		return err
	}
	for _, specPatch := range version.Data.ApiSpecs {
		err := applyApiSpecPatch(ctx, client, specPatch, name.String(), "")
		if err != nil {
			return err
		}
	}
	for _, artifactPatch := range version.Data.Artifacts {
		err = applyArtifactPatch(ctx, client, artifactPatch, name.String())
		if err != nil {
			return err
		}
	}
	return nil
}
