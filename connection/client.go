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

package connection

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/apigee/registry/gapic"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// Settings configure the client.
type Settings struct {
	Address  string // service address
	Insecure bool   // if true, connect over HTTP
	Token    string // bearer token
}

func newSettings() (*Settings, error) {
	settings := &Settings{}
	settings.Address = os.Getenv("APG_REGISTRY_ADDRESS")
	if settings.Address == "" {
		return nil, fmt.Errorf("rpc error: APG_REGISTRY_ADDRESS must be set")
	}
	settings.Insecure, _ = strconv.ParseBool(os.Getenv("APG_REGISTRY_INSECURE"))
	settings.Token = os.Getenv("APG_REGISTRY_TOKEN")
	return settings, nil
}

func clientOptions(settings *Settings) ([]option.ClientOption, error) {
	var opts []option.ClientOption
	if settings.Address == "" {
		return nil, fmt.Errorf("rpc error: address must be set")
	}
	opts = append(opts, option.WithEndpoint(settings.Address))
	if settings.Insecure {
		conn, err := grpc.Dial(settings.Address, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		opts = append(opts, option.WithGRPCConn(conn))
	}
	if settings.Token != "" {
		opts = append(opts, option.WithTokenSource(oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: settings.Token,
				TokenType:   "Bearer",
			})))
	}
	return opts, nil
}

// Client is a client of the Registry API
// TODO: Rename this to RegistryClient
type Client = *gapic.RegistryClient

// NewClient creates a new GAPIC client using environment variable settings.
func NewClient(ctx context.Context) (Client, error) {
	settings, err := newSettings()
	if err != nil {
		return nil, err
	}
	return NewClientWithSettings(ctx, settings)
}

// NewClientWithSettings creates a GAPIC client with specified settings.
func NewClientWithSettings(ctx context.Context, settings *Settings) (Client, error) {
	opts, err := clientOptions(settings)
	if err != nil {
		return nil, err
	}
	return gapic.NewRegistryClient(ctx, opts...)
}

type AdminClient = *gapic.AdminClient

// NewAdminClient creates a new GAPIC client using environment variable settings.
func NewAdminClient(ctx context.Context) (AdminClient, error) {
	settings, err := newSettings()
	if err != nil {
		return nil, err
	}
	return NewAdminClientWithSettings(ctx, settings)
}

// NewAdminClientWithSettings creates a GAPIC client with specified settings.
func NewAdminClientWithSettings(ctx context.Context, settings *Settings) (AdminClient, error) {
	opts, err := clientOptions(settings)
	if err != nil {
		return nil, err
	}
	return gapic.NewAdminClient(ctx, opts...)
}
