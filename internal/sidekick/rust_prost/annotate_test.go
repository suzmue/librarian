// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust_prost

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestModelAnnotations(t *testing.T) {
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		SpecificationSource: "../../testdata/googleapis/google/type",
		Source: &sources.SourceConfig{
			IncludeList: []string{"f1.proto", "f2.proto"},
		},
		Codec: map[string]string{
			"copyright-year": "2035",
		},
	}
	model := api.NewTestAPI(
		nil, nil,
		[]*api.Service{api.NewTestService("Workflows").WithPackage("google.cloud.workflows.v1")})
	codec := newCodec(cfg)
	if err := codec.annotateModel(model, cfg); err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		PackageName:   "google-cloud-workflows-v1",
		CopyrightYear: "2035",
		Files: []string{
			"../../testdata/googleapis/google/type/f1.proto",
			"../../testdata/googleapis/google/type/f2.proto",
		},
	}
	if diff := cmp.Diff(want, model.Codec, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceAnnotations(t *testing.T) {
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		SpecificationSource: "../../testdata/googleapis/google/type",
		Source: &sources.SourceConfig{
			IncludeList: []string{"unused.proto"},
		},
		Codec: map[string]string{
			"copyright-year": "2035",
		},
	}
	model := api.NewTestAPI(
		nil, nil,
		[]*api.Service{
			api.NewTestService("Workflows").WithPackage("google.cloud.workflows.v1"),
		})
	codec := newCodec(cfg)
	if err := codec.annotateModel(model, cfg); err != nil {
		t.Fatal(err)
	}
	want := &serviceAnnotations{
		ID: "google.cloud.workflows.v1.Workflows",
	}
	got := model.Service(".google.cloud.workflows.v1.Workflows")
	if got == nil {
		t.Fatalf("cannot find service %s", ".google.cloud.workflows.v1.Workflows")
	}
	if diff := cmp.Diff(want, got.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethodAnnotations(t *testing.T) {
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		SpecificationSource: "../../testdata/googleapis/google/type",
		Source: &sources.SourceConfig{
			IncludeList: []string{"unused.proto"},
		},
		Codec: map[string]string{
			"copyright-year": "2035",
		},
	}
	model := api.NewTestAPI(
		nil, nil,
		[]*api.Service{
			api.NewTestService("Workflows").
				WithPackage("google.cloud.workflows.v1").
				WithMethods(api.NewTestMethod("GetWorkflow")),
		})
	codec := newCodec(cfg)
	if err := codec.annotateModel(model, cfg); err != nil {
		t.Fatal(err)
	}
	want := &methodAnnotations{
		ID: "google.cloud.workflows.v1.Workflows.GetWorkflow",
	}
	got := model.Method(".google.cloud.workflows.v1.Workflows.GetWorkflow")
	if got == nil {
		t.Fatalf("cannot find service %s", ".google.cloud.workflows.v1.Workflows.GetWorkflow")
	}
	if diff := cmp.Diff(want, got.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestModelAnnotationsMixinFiles(t *testing.T) {
	for _, test := range []struct {
		name        string
		includeList []string
		service     *api.Service
		unusedTypes string
		wantFiles   []string
	}{
		{
			name:        "all mixins used",
			includeList: []string{"f1.proto"},
			service: api.NewTestService("Workflows").
				WithPackage("google.cloud.workflows.v1").
				WithMethods(
					api.NewTestMethod("GetLocation").
						WithSourceServiceID(".google.cloud.location.Locations").
						WithInputTypeID(".google.cloud.location.GetLocationRequest"),
					api.NewTestMethod("ListLocations").
						WithSourceServiceID(".google.cloud.location.Locations").
						WithInputTypeID(".google.cloud.location.ListLocationsRequest"),
					api.NewTestMethod("GetIamPolicy").
						WithSourceServiceID(".google.iam.v1.IAMPolicy").
						WithInputTypeID(".google.iam.v1.GetIamPolicyRequest"),
					api.NewTestMethod("GetOperation").
						WithSourceServiceID(".google.longrunning.Operations").
						WithInputTypeID(".google.longrunning.GetOperationRequest"),
				),
			wantFiles: []string{
				"../../testdata/googleapis/google/type/f1.proto",
				"google/cloud/location/locations.proto",
				"google/iam/v1/iam_policy.proto",
				"google/longrunning/operations.proto",
			},
		},
		{
			name:        "unused mixin types skipped",
			includeList: []string{"f1.proto"},
			service: api.NewTestService("Workflows").
				WithPackage("google.cloud.workflows.v1").
				WithMethods(
					api.NewTestMethod("GetLocation").
						WithSourceServiceID(".google.cloud.location.Locations").
						WithInputTypeID(".google.cloud.location.GetLocationRequest"),
					api.NewTestMethod("GetOperation").
						WithSourceServiceID(".google.longrunning.Operations").
						WithInputTypeID(".google.longrunning.GetOperationRequest"),
				),
			unusedTypes: ".google.cloud.location.GetLocationRequest",
			wantFiles: []string{
				"../../testdata/googleapis/google/type/f1.proto",
				"google/longrunning/operations.proto",
			},
		},
		{
			name:        "same service ID is not treated as mixin",
			includeList: []string{"google/cloud/location/locations.proto"},
			service: api.NewTestService("Locations").
				WithPackage("google.cloud.location").
				WithMethods(
					api.NewTestMethod("GetLocation").
						WithSourceServiceID(".google.cloud.location.Locations").
						WithInputTypeID(".google.cloud.location.GetLocationRequest"),
				),
			wantFiles: []string{
				"../../testdata/googleapis/google/type/google/cloud/location/locations.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				SpecificationSource: "../../testdata/googleapis/google/type",
				Source: &sources.SourceConfig{
					IncludeList: test.includeList,
				},
				Codec: map[string]string{
					"copyright-year": "2035",
					"unused-types":   test.unusedTypes,
				},
			}
			model := api.NewTestAPI(nil, nil, []*api.Service{test.service})
			codec := newCodec(cfg)
			if err := codec.annotateModel(model, cfg); err != nil {
				t.Fatal(err)
			}
			ann, ok := model.Codec.(*modelAnnotations)
			if !ok {
				t.Fatalf("unexpected codec type %T", model.Codec)
			}
			if diff := cmp.Diff(test.wantFiles, ann.Files); diff != "" {
				t.Errorf("Files mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
