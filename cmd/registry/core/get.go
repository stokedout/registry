// Copyright 2020 Google LLC. All Rights Reserved.
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

package core

import (
	"context"

	"github.com/apigee/registry/gapic"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/names"
	"google.golang.org/grpc/metadata"
)

func GetProject(ctx context.Context,
	client *gapic.AdminClient,
	name names.Project,
	handler ProjectHandler) error {
	project, err := client.GetProject(ctx, &rpc.GetProjectRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}

	return handler(project)
}

func GetAPI(ctx context.Context,
	client *gapic.RegistryClient,
	name names.Api,
	handler ApiHandler) error {
	api, err := client.GetApi(ctx, &rpc.GetApiRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}

	return handler(api)
}

func GetDeployment(ctx context.Context,
	client *gapic.RegistryClient,
	name names.Deployment,
	handler DeploymentHandler) error {
	deployment, err := client.GetApiDeployment(ctx, &rpc.GetApiDeploymentRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}

	return handler(deployment)
}

func GetDeploymentRevision(ctx context.Context,
	client *gapic.RegistryClient,
	name names.DeploymentRevision,
	handler DeploymentHandler) error {
	request := &rpc.GetApiDeploymentRequest{
		Name: name.String(),
	}
	deployment, err := client.GetApiDeployment(ctx, request)
	if err != nil {
		return err
	}

	return handler(deployment)
}

func GetVersion(ctx context.Context,
	client *gapic.RegistryClient,
	name names.Version,
	handler VersionHandler) error {
	version, err := client.GetApiVersion(ctx, &rpc.GetApiVersionRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}

	return handler(version)
}

func GetSpec(ctx context.Context,
	client *gapic.RegistryClient,
	name names.Spec,
	getContents bool,
	handler SpecHandler) error {
	spec, err := client.GetApiSpec(ctx, &rpc.GetApiSpecRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}
	if getContents {
		if err := FetchSpecContents(ctx, client, spec); err != nil {
			return err
		}
	}

	return handler(spec)
}

func GetSpecRevision(ctx context.Context,
	client *gapic.RegistryClient,
	name names.SpecRevision,
	getContents bool,
	handler SpecHandler) error {
	request := &rpc.GetApiSpecRequest{
		Name: name.String(),
	}
	spec, err := client.GetApiSpec(ctx, request)
	if err != nil {
		return err
	}
	if getContents {
		if err := FetchSpecContents(ctx, client, spec); err != nil {
			return err
		}
	}

	return handler(spec)
}

func GetArtifact(ctx context.Context,
	client *gapic.RegistryClient,
	name names.Artifact,
	getContents bool,
	handler ArtifactHandler) error {
	artifact, err := client.GetArtifact(ctx, &rpc.GetArtifactRequest{
		Name: name.String(),
	})
	if err != nil {
		return err
	}
	if getContents {
		if err = FetchArtifactContents(ctx, client, artifact); err != nil {
			return err
		}
	}

	return handler(artifact)
}

func FetchSpecContents(ctx context.Context, client *gapic.RegistryClient, spec *rpc.ApiSpec) error {
	if spec.Contents != nil {
		return nil
	}
	request := &rpc.GetApiSpecContentsRequest{
		Name: spec.GetName(),
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "accept-encoding", "gzip")
	contents, err := client.GetApiSpecContents(ctx, request)
	if err != nil {
		return err
	}
	spec.Contents = contents.GetData()
	spec.MimeType = contents.GetContentType()
	return nil
}

func FetchArtifactContents(ctx context.Context, client *gapic.RegistryClient, artifact *rpc.Artifact) error {
	if artifact.Contents != nil {
		return nil
	}
	contents, err := client.GetArtifactContents(ctx, &rpc.GetArtifactContentsRequest{
		Name: artifact.GetName(),
	})
	if err != nil {
		return err
	}
	artifact.Contents = contents.GetData()
	artifact.MimeType = contents.GetContentType()
	return nil
}
