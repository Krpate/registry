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

package upload

import (
	"context"

	"github.com/apigee/registry/cmd/registry/cmd/upload/bulk"
	"github.com/spf13/cobra"
)

func Command(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload information to the API Registry",
	}

	cmd.AddCommand(bulk.Command(ctx))
	cmd.AddCommand(csvCommand(ctx))
	cmd.AddCommand(manifestCommand(ctx))
	cmd.AddCommand(specCommand(ctx))
	cmd.AddCommand(styleGuideCommand(ctx))

	return cmd
}
