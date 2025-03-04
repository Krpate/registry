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

package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/internal/test/seeder"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestCreateApi(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Project
		req  *rpc.CreateApiRequest
		want *rpc.Api
	}{
		{
			desc: "fully populated resource",
			seed: &rpc.Project{
				Name: "projects/my-project",
			},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "my-api",
				Api: &rpc.Api{
					DisplayName:        "My Display Name",
					Description:        "My Description",
					Availability:       "My Availability",
					RecommendedVersion: "My Version",
					Labels: map[string]string{
						"label-key": "label-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			want: &rpc.Api{
				Name:               "projects/my-project/locations/global/apis/my-api",
				DisplayName:        "My Display Name",
				Description:        "My Description",
				Availability:       "My Availability",
				RecommendedVersion: "My Version",
				Labels: map[string]string{
					"label-key": "label-value",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedProjects(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			created, err := server.CreateApi(ctx, test.req)
			if err != nil {
				t.Fatalf("CreateApi(%+v) returned error: %s", test.req, err)
			}

			opts := cmp.Options{
				protocmp.Transform(),
				protocmp.IgnoreFields(new(rpc.Api), "create_time", "update_time"),
			}

			if !cmp.Equal(test.want, created, opts) {
				t.Errorf("CreateApi(%+v) returned unexpected diff (-want +got):\n%s", test.req, cmp.Diff(test.want, created, opts))
			}

			if created.CreateTime == nil || created.UpdateTime == nil {
				t.Errorf("CreateApi(%+v) returned unset create_time (%v) or update_time (%v)", test.req, created.CreateTime, created.UpdateTime)
			} else if !created.CreateTime.AsTime().Equal(created.UpdateTime.AsTime()) {
				t.Errorf("CreateApi(%+v) returned unexpected timestamps: create_time %v != update_time %v", test.req, created.CreateTime, created.UpdateTime)
			}

			t.Run("GetApi", func(t *testing.T) {
				req := &rpc.GetApiRequest{
					Name: created.GetName(),
				}

				got, err := server.GetApi(ctx, req)
				if err != nil {
					t.Fatalf("GetApi(%+v) returned error: %s", req, err)
				}

				opts := protocmp.Transform()
				if !cmp.Equal(created, got, opts) {
					t.Errorf("GetApi(%+v) returned unexpected diff (-want +got):\n%s", req, cmp.Diff(created, got, opts))
				}
			})
		})
	}
}

func TestCreateApiResponseCodes(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Project
		req  *rpc.CreateApiRequest
		want codes.Code
	}{
		{
			desc: "parent not found",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/other-project/locations/global",
				ApiId:  "valid-id",
				Api:    &rpc.Api{},
			},
			want: codes.NotFound,
		},
		{
			desc: "missing resource body",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "valid-id",
				Api:    nil,
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "missing custom identifier",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "long custom identifier",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "this-identifier-is-invalid-because-it-exceeds-the-eighty-character-maximum-length",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "custom identifier underscores",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "underscore_identifier",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "custom identifier hyphen prefix",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "-identifier",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "custom identifier hyphen suffix",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "identifier-",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "customer identifier uuid format",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "072d2288-c685-42d8-9df0-5edbb2a809ea",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "custom identifier mixed case",
			seed: &rpc.Project{Name: "projects/my-project"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "IDentifier",
				Api:    &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedProjects(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			if _, err := server.CreateApi(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("CreateApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}

func TestCreateApiDuplicates(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.CreateApiRequest
		want codes.Code
	}{
		{
			desc: "case sensitive",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "my-api",
				Api:    &rpc.Api{},
			},
			want: codes.AlreadyExists,
		},
		{
			desc: "case insensitive",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.CreateApiRequest{
				Parent: "projects/my-project/locations/global",
				ApiId:  "My-Api",
				Api:    &rpc.Api{},
			},
			want: codes.AlreadyExists,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			if _, err := server.CreateApi(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("CreateApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}

func TestGetApi(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.GetApiRequest
		want *rpc.Api
	}{
		{
			desc: "fully populated resource",
			seed: &rpc.Api{
				Name:               "projects/my-project/locations/global/apis/my-api",
				DisplayName:        "My Display Name",
				Description:        "My Description",
				Availability:       "My Availability",
				RecommendedVersion: "My Version",
				Labels: map[string]string{
					"label-key": "label-value",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			},
			req: &rpc.GetApiRequest{
				Name: "projects/my-project/locations/global/apis/my-api",
			},
			want: &rpc.Api{
				Name:               "projects/my-project/locations/global/apis/my-api",
				DisplayName:        "My Display Name",
				Description:        "My Description",
				Availability:       "My Availability",
				RecommendedVersion: "My Version",
				Labels: map[string]string{
					"label-key": "label-value",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			got, err := server.GetApi(ctx, test.req)
			if err != nil {
				t.Fatalf("GetApi(%+v) returned error: %s", test.req, err)
			}

			opts := cmp.Options{
				protocmp.Transform(),
				protocmp.IgnoreFields(new(rpc.Api), "create_time", "update_time"),
			}

			if !cmp.Equal(test.want, got, opts) {
				t.Errorf("GetApi(%+v) returned unexpected diff (-want +got):\n%s", test.req, cmp.Diff(test.want, got, opts))
			}
		})
	}
}

func TestGetApiResponseCodes(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.GetApiRequest
		want codes.Code
	}{
		{
			desc: "resource not found",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.GetApiRequest{
				Name: "projects/my-project/locations/global/apis/doesnt-exist",
			},
			want: codes.NotFound,
		},
		{
			desc: "case insensitive name",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.GetApiRequest{
				Name: "projects/my-project/locations/global/apis/My-Api",
			},
			want: codes.OK,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			if _, err := server.GetApi(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("GetApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}

func TestListApis(t *testing.T) {
	tests := []struct {
		desc      string
		seed      []*rpc.Api
		req       *rpc.ListApisRequest
		want      *rpc.ListApisResponse
		wantToken bool
		extraOpts cmp.Option
	}{
		{
			desc: "default parameters",
			seed: []*rpc.Api{
				{Name: "projects/my-project/locations/global/apis/api1"},
				{Name: "projects/my-project/locations/global/apis/api2"},
				{Name: "projects/my-project/locations/global/apis/api3"},
				{Name: "projects/other-project/locations/global/apis/api1"},
			},
			req: &rpc.ListApisRequest{
				Parent: "projects/my-project/locations/global",
			},
			want: &rpc.ListApisResponse{
				Apis: []*rpc.Api{
					{Name: "projects/my-project/locations/global/apis/api1"},
					{Name: "projects/my-project/locations/global/apis/api2"},
					{Name: "projects/my-project/locations/global/apis/api3"},
				},
			},
		},
		{
			desc: "across all projects",
			seed: []*rpc.Api{
				{Name: "projects/my-project/locations/global/apis/api1"},
				{Name: "projects/my-project/locations/global/apis/api2"},
				{Name: "projects/my-project/locations/global/apis/api3"},
				{Name: "projects/other-project/locations/global/apis/api1"},
			},
			req: &rpc.ListApisRequest{
				Parent: "projects/-/locations/global",
			},
			want: &rpc.ListApisResponse{
				Apis: []*rpc.Api{
					{Name: "projects/my-project/locations/global/apis/api1"},
					{Name: "projects/my-project/locations/global/apis/api2"},
					{Name: "projects/my-project/locations/global/apis/api3"},
					{Name: "projects/other-project/locations/global/apis/api1"},
				},
			},
		},
		{
			desc: "custom page size",
			seed: []*rpc.Api{
				{Name: "projects/my-project/locations/global/apis/api1"},
				{Name: "projects/my-project/locations/global/apis/api2"},
				{Name: "projects/my-project/locations/global/apis/api3"},
			},
			req: &rpc.ListApisRequest{
				Parent:   "projects/my-project/locations/global",
				PageSize: 1,
			},
			want: &rpc.ListApisResponse{
				Apis: []*rpc.Api{
					{},
				},
			},
			wantToken: true,
			// Ordering is not guaranteed by API, so any resource may be returned.
			extraOpts: protocmp.IgnoreFields(new(rpc.Api), "name"),
		},
		{
			desc: "name equality filtering",
			seed: []*rpc.Api{
				{Name: "projects/my-project/locations/global/apis/api1"},
				{Name: "projects/my-project/locations/global/apis/api2"},
				{Name: "projects/my-project/locations/global/apis/api3"},
			},
			req: &rpc.ListApisRequest{
				Parent: "projects/my-project/locations/global",
				Filter: "name == 'projects/my-project/locations/global/apis/api2'",
			},
			want: &rpc.ListApisResponse{
				Apis: []*rpc.Api{
					{Name: "projects/my-project/locations/global/apis/api2"},
				},
			},
		},
		{
			desc: "description inequality filtering",
			seed: []*rpc.Api{
				{
					Name:        "projects/my-project/locations/global/apis/api1",
					Description: "First Api",
				},
				{Name: "projects/my-project/locations/global/apis/api2"},
				{Name: "projects/my-project/locations/global/apis/api3"},
			},
			req: &rpc.ListApisRequest{
				Parent: "projects/my-project/locations/global",
				Filter: "description != ''",
			},
			want: &rpc.ListApisResponse{
				Apis: []*rpc.Api{
					{
						Name:        "projects/my-project/locations/global/apis/api1",
						Description: "First Api",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed...); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			got, err := server.ListApis(ctx, test.req)
			if err != nil {
				t.Fatalf("ListApis(%+v) returned error: %s", test.req, err)
			}

			opts := cmp.Options{
				protocmp.Transform(),
				protocmp.IgnoreFields(new(rpc.ListApisResponse), "next_page_token"),
				protocmp.IgnoreFields(new(rpc.Api), "create_time", "update_time"),
				protocmp.SortRepeated(func(a, b *rpc.Api) bool {
					return a.GetName() < b.GetName()
				}),
				test.extraOpts,
			}

			if !cmp.Equal(test.want, got, opts) {
				t.Errorf("ListApis(%+v) returned unexpected diff (-want +got):\n%s", test.req, cmp.Diff(test.want, got, opts))
			}

			if test.wantToken && got.NextPageToken == "" {
				t.Errorf("ListApis(%+v) returned empty next_page_token, expected non-empty next_page_token", test.req)
			} else if !test.wantToken && got.NextPageToken != "" {
				t.Errorf("ListApis(%+v) returned non-empty next_page_token, expected empty next_page_token: %s", test.req, got.GetNextPageToken())
			}
		})
	}
}

func TestListApisResponseCodes(t *testing.T) {
	tests := []struct {
		desc string
		req  *rpc.ListApisRequest
		want codes.Code
	}{
		{
			desc: "parent not found",
			req: &rpc.ListApisRequest{
				Parent: "projects/my-project/locations/global",
			},
			want: codes.NotFound,
		},
		{
			desc: "negative page size",
			req: &rpc.ListApisRequest{
				PageSize: -1,
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "invalid filter",
			req: &rpc.ListApisRequest{
				Filter: "this filter is not valid",
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "invalid page token",
			req: &rpc.ListApisRequest{
				PageToken: "this token is not valid",
			},
			want: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)

			if _, err := server.ListApis(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("ListApis(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}

func TestListApisSequence(t *testing.T) {
	ctx := context.Background()
	server := defaultTestServer(t)
	seed := []*rpc.Api{
		{Name: "projects/my-project/locations/global/apis/api1"},
		{Name: "projects/my-project/locations/global/apis/api2"},
		{Name: "projects/my-project/locations/global/apis/api3"},
	}
	if err := seeder.SeedApis(ctx, server, seed...); err != nil {
		t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
	}

	listed := make([]*rpc.Api, 0, 3)

	var nextToken string
	t.Run("first page", func(t *testing.T) {
		req := &rpc.ListApisRequest{
			Parent:   "projects/my-project/locations/global",
			PageSize: 1,
		}

		got, err := server.ListApis(ctx, req)
		if err != nil {
			t.Fatalf("ListApis(%+v) returned error: %s", req, err)
		}

		if count := len(got.GetApis()); count != 1 {
			t.Errorf("ListApis(%+v) returned %d apis, expected exactly one", req, count)
		}

		if got.GetNextPageToken() == "" {
			t.Errorf("ListApis(%+v) returned empty next_page_token, expected another page", req)
		}

		listed = append(listed, got.Apis...)
		nextToken = got.GetNextPageToken()
	})

	if t.Failed() {
		t.Fatal("Cannot test intermediate page after failure on first page")
	}

	t.Run("intermediate page", func(t *testing.T) {
		req := &rpc.ListApisRequest{
			Parent:    "projects/my-project/locations/global",
			PageSize:  1,
			PageToken: nextToken,
		}

		got, err := server.ListApis(ctx, req)
		if err != nil {
			t.Fatalf("ListApis(%+v) returned error: %s", req, err)
		}

		if count := len(got.GetApis()); count != 1 {
			t.Errorf("ListApis(%+v) returned %d apis, expected exactly one", req, count)
		}

		if got.GetNextPageToken() == "" {
			t.Errorf("ListApis(%+v) returned empty next_page_token, expected another page", req)
		}

		listed = append(listed, got.Apis...)
		nextToken = got.GetNextPageToken()
	})

	if t.Failed() {
		t.Fatal("Cannot test final page after failure on intermediate page")
	}

	t.Run("final page", func(t *testing.T) {
		req := &rpc.ListApisRequest{
			Parent:    "projects/my-project/locations/global",
			PageSize:  1,
			PageToken: nextToken,
		}

		got, err := server.ListApis(ctx, req)
		if err != nil {
			t.Fatalf("ListApis(%+v) returned error: %s", req, err)
		}

		if count := len(got.GetApis()); count != 1 {
			t.Errorf("ListApis(%+v) returned %d apis, expected exactly one", req, count)
		}

		if got.GetNextPageToken() != "" {
			t.Errorf("ListApis(%+v) returned next_page_token, expected no next page", req)
		}

		listed = append(listed, got.Apis...)
	})

	if t.Failed() {
		t.Fatal("Cannot test sequence result after failure on final page")
	}

	opts := cmp.Options{
		protocmp.Transform(),
		protocmp.IgnoreFields(new(rpc.Api), "create_time", "update_time"),
		cmpopts.SortSlices(func(a, b *rpc.Api) bool {
			return a.GetName() < b.GetName()
		}),
	}

	if !cmp.Equal(seed, listed, opts) {
		t.Errorf("List sequence returned unexpected diff (-want +got):\n%s", cmp.Diff(seed, listed, opts))
	}
}

// This test prevents the list sequence from ending before a known filter match is listed.
// For simplicity, it does not guarantee the resource is returned on a later page.
func TestListApisLargeCollectionFiltering(t *testing.T) {
	ctx := context.Background()
	server := defaultTestServer(t)
	seed := make([]*rpc.Api, 0, 100)
	for i := 1; i <= cap(seed); i++ {
		seed = append(seed, &rpc.Api{
			Name: fmt.Sprintf("projects/my-project/locations/global/apis/a%03d", i),
		})
	}

	if err := seeder.SeedApis(ctx, server, seed...); err != nil {
		t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
	}

	req := &rpc.ListApisRequest{
		Parent:   "projects/my-project/locations/global",
		PageSize: 1,
		Filter:   "name == 'projects/my-project/locations/global/apis/a099'",
	}

	got, err := server.ListApis(ctx, req)
	if err != nil {
		t.Fatalf("ListApis(%+v) returned error: %s", req, err)
	}

	if len(got.GetApis()) == 1 && got.GetNextPageToken() != "" {
		t.Errorf("ListApis(%+v) returned a page token when the only matching resource has been listed: %+v", req, got)
	} else if len(got.GetApis()) == 0 && got.GetNextPageToken() == "" {
		t.Errorf("ListApis(%+v) returned an empty next page token before listing the only matching resource", req)
	} else if count := len(got.GetApis()); count > 1 {
		t.Errorf("ListApis(%+v) returned %d projects, expected at most one: %+v", req, count, got.GetApis())
	}
}

func TestUpdateApi(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.UpdateApiRequest
		want *rpc.Api
	}{
		{
			desc: "allow missing updates existing resources",
			seed: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/a",
				Description: "My Api",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name:        "projects/my-project/locations/global/apis/a",
					Description: "My Updated Api",
				},
				UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				AllowMissing: true,
			},
			want: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/a",
				Description: "My Updated Api",
			},
		},
		{
			desc: "allow missing creates missing resources",
			seed: &rpc.Api{
				Name: "projects/my-project/locations/global/apis/a-sibling",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name: "projects/my-project/locations/global/apis/a",
				},
				AllowMissing: true,
			},
			want: &rpc.Api{
				Name: "projects/my-project/locations/global/apis/a",
			},
		},
		{
			desc: "implicit nil mask",
			seed: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Api",
				Description: "Api for my APIs",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name:        "projects/my-project/locations/global/apis/my-api",
					DisplayName: "My Updated Api",
				},
			},
			want: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Updated Api",
				Description: "Api for my APIs",
			},
		},
		{
			desc: "implicit empty mask",
			seed: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Api",
				Description: "Api for my APIs",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name:        "projects/my-project/locations/global/apis/my-api",
					DisplayName: "My Updated Api",
				},
				UpdateMask: &fieldmaskpb.FieldMask{},
			},
			want: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Updated Api",
				Description: "Api for my APIs",
			},
		},
		{
			desc: "field specific mask",
			seed: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Api",
				Description: "Api for my APIs",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name:        "projects/my-project/locations/global/apis/my-api",
					DisplayName: "My Updated Api",
					Description: "Ignored",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
			},
			want: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Updated Api",
				Description: "Api for my APIs",
			},
		},
		{
			desc: "full replacement wildcard mask",
			seed: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Api",
				Description: "Api for my APIs",
			},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name:        "projects/my-project/locations/global/apis/my-api",
					DisplayName: "My Updated Api",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"*"}},
			},
			want: &rpc.Api{
				Name:        "projects/my-project/locations/global/apis/my-api",
				DisplayName: "My Updated Api",
				Description: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			updated, err := server.UpdateApi(ctx, test.req)
			if err != nil {
				t.Fatalf("UpdateApi(%+v) returned error: %s", test.req, err)
			}

			opts := cmp.Options{
				protocmp.Transform(),
				protocmp.IgnoreFields(new(rpc.Api), "create_time", "update_time"),
			}

			if !cmp.Equal(test.want, updated, opts) {
				t.Errorf("UpdateApi(%+v) returned unexpected diff (-want +got):\n%s", test.req, cmp.Diff(test.want, updated, opts))
			}

			t.Run("GetApi", func(t *testing.T) {
				req := &rpc.GetApiRequest{
					Name: updated.GetName(),
				}

				got, err := server.GetApi(ctx, req)
				if err != nil {
					t.Fatalf("GetApi(%+v) returned error: %s", req, err)
				}

				opts := protocmp.Transform()
				if !cmp.Equal(updated, got, opts) {
					t.Errorf("GetApi(%+v) returned unexpected diff (-want +got):\n%s", req, cmp.Diff(updated, got, opts))
				}
			})
		})
	}
}

func TestUpdateApiResponseCodes(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.UpdateApiRequest
		want codes.Code
	}{
		{
			desc: "resource not found",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name: "projects/my-project/locations/global/apis/doesnt-exist",
				},
			},
			want: codes.NotFound,
		},
		{
			desc: "missing resource body",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req:  &rpc.UpdateApiRequest{},
			want: codes.InvalidArgument,
		},
		{
			desc: "missing resource name",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{},
			},
			want: codes.InvalidArgument,
		},
		{
			desc: "nonexistent field in mask",
			seed: &rpc.Api{Name: "projects/my-project/locations/global/apis/my-api"},
			req: &rpc.UpdateApiRequest{
				Api: &rpc.Api{
					Name: "projects/my-project/locations/global/apis/my-api",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"this field does not exist"}},
			},
			want: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			if _, err := server.UpdateApi(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("UpdateApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}

func TestDeleteApi(t *testing.T) {
	tests := []struct {
		desc string
		seed *rpc.Api
		req  *rpc.DeleteApiRequest
	}{
		{
			desc: "existing resource",
			seed: &rpc.Api{
				Name: "projects/my-project/locations/global/apis/my-api",
			},
			req: &rpc.DeleteApiRequest{
				Name: "projects/my-project/locations/global/apis/my-api",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)
			if err := seeder.SeedApis(ctx, server, test.seed); err != nil {
				t.Fatalf("Setup/Seeding: Failed to seed registry: %s", err)
			}

			if _, err := server.DeleteApi(ctx, test.req); err != nil {
				t.Fatalf("DeleteApi(%+v) returned error: %s", test.req, err)
			}

			t.Run("GetApi", func(t *testing.T) {
				req := &rpc.GetApiRequest{
					Name: test.req.GetName(),
				}

				if _, err := server.GetApi(ctx, req); status.Code(err) != codes.NotFound {
					t.Fatalf("GetApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), codes.NotFound, err)
				}
			})
		})
	}
}

func TestDeleteApiResponseCodes(t *testing.T) {
	tests := []struct {
		desc string
		req  *rpc.DeleteApiRequest
		want codes.Code
	}{
		{
			desc: "resource not found",
			req: &rpc.DeleteApiRequest{
				Name: "projects/my-project/locations/global/apis/doesnt-exist",
			},
			want: codes.NotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			server := defaultTestServer(t)

			if _, err := server.DeleteApi(ctx, test.req); status.Code(err) != test.want {
				t.Errorf("DeleteApi(%+v) returned status code %q, want %q: %v", test.req, status.Code(err), test.want, err)
			}
		})
	}
}
