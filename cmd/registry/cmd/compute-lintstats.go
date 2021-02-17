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

package cmd

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/apigee/registry/cmd/registry/core"
	"github.com/apigee/registry/connection"
	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/names"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func init() {
	computeCmd.AddCommand(computeLintStatsCmd)
	computeLintStatsCmd.Flags().String("linter", "", "name of linter associated with these lintstats (aip, spectral, gnostic)")
}

func lintStatsRelation(linter string) string {
	return "lintstats-" + linter
}

var computeLintStatsCmd = &cobra.Command{
	Use:   "lintstats",
	Short: "Compute summaries of linter runs",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		linter, err := cmd.LocalFlags().GetString("linter")
		if err != nil || linter == "" {
			log.Fatalf("Please specify a linter with the --linter flag")
		}

		ctx := context.TODO()
		client, err := connection.NewClient(ctx)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		// Generate tasks.
		name := args[0]
		if m := names.SpecRegexp().FindStringSubmatch(name); m != nil {
			// Iterate through a collection of specs and evaluate each.
			err = core.ListSpecs(ctx, client, m, computeFilter, func(spec *rpc.ApiSpec) {
				fmt.Printf("%s\n", spec.Name)
				// get the lint results
				request := rpc.GetArtifactRequest{
					Name: spec.Name + "/artifacts/" + lintRelation(linter),
					View: rpc.View_FULL,
				}
				artifact, err := client.GetArtifact(ctx, &request)
				if artifact == nil {
					return // ignore missing results
				}
				if artifact.GetMimeType() != "google.cloud.apigee.registry.v1alpha1.Lint" {
					return // ignore unexpected message types
				}
				lint := &rpc.Lint{}
				err = proto.Unmarshal(artifact.GetContents(), lint)
				if err != nil {
					log.Printf("%+v", err)
					return
				}
				// generate the stats from the result by counting problems
				lintStats := computeLintStats(lint)
				{
					// store the lintstats artifact
					subject := spec.GetName()
					relation := lintStatsRelation(linter)
					messageData, err := proto.Marshal(lintStats)
					artifact := &rpc.Artifact{
						Name:     subject + "/artifacts/" + relation,
						MimeType: "google.cloud.apigee.registry.v1alpha1.LintStats",
						Contents: messageData,
					}
					err = core.SetArtifact(ctx, client, artifact)
					if err != nil {
						log.Printf("%+v", err)
						return
					}
				}
			})
			if err != nil {
				log.Fatalf("%s", err.Error())
			}
		}
		if m := names.ProjectRegexp().FindStringSubmatch(name); m != nil {
			// Iterate through a collection of projects and evaluate each.
			err = core.ListProjects(ctx, client, m, computeFilter, func(project *rpc.Project) {
				// Create a top-level list of problem counts for the project
				problemCounts := make([]*rpc.LintProblemCount, 0)
				// get the lintstats for each spec in the project
				pattern := project.Name + "/apis/-/versions/-/specs/-/artifacts/" + lintStatsRelation(linter)
				if m2 := names.ArtifactRegexp().FindStringSubmatch(pattern); m2 != nil {
					err = core.ListArtifacts(ctx, client, m2, "", true, func(artifact *rpc.Artifact) {
						log.Printf("%+v", artifact.Name)
						// get the lintstats artifact value
						if artifact.GetMimeType() != "google.cloud.apigee.registry.v1alpha1.LintStats" {
							return // ignore unexpected message types
						}
						lintstats := &rpc.LintStats{}
						err = proto.Unmarshal(artifact.GetContents(), lintstats)
						if err != nil {
							log.Printf("%+v", err)
							return
						}
						// merge the lintstats into the problemCounts slice
						problemCounts = mergeLintStats(problemCounts, lintstats)
					})
				}
				// sort results in decreasing order of count
				sort.Slice(problemCounts, func(i, j int) bool {
					return problemCounts[i].Count > problemCounts[j].Count
				})
				// store the summary in the lintstats artifact
				lintstats := &rpc.LintStats{ProblemCounts: problemCounts}
				{
					// store the lintstats artifact
					subject := project.GetName()
					relation := lintStatsRelation(linter)
					messageData, err := proto.Marshal(lintstats)
					artifact := &rpc.Artifact{
						Name:     subject + "/artifacts/" + relation,
						MimeType: "google.cloud.apigee.registry.v1alpha1.LintStats",
						Contents: messageData,
					}
					err = core.SetArtifact(ctx, client, artifact)
					if err != nil {
						log.Printf("%+v", err)
						return
					}
				}
			})
			if err != nil {
				log.Fatalf("%s", err.Error())
			}
		}
	},
}

func computeLintStats(lint *rpc.Lint) *rpc.LintStats {
	problemCounts := make([]*rpc.LintProblemCount, 0)
	for _, file := range lint.Files {
		for _, problem := range file.Problems {
			var problemCount *rpc.LintProblemCount
			for _, pc := range problemCounts {
				if pc.RuleId == problem.RuleId {
					problemCount = pc
					break
				}
			}
			if problemCount == nil {
				problemCount = &rpc.LintProblemCount{
					Count:      0,
					RuleId:     problem.RuleId,
					RuleDocUri: problem.RuleDocUri,
				}
				problemCounts = append(problemCounts, problemCount)
			}
			problemCount.Count++
		}
	}
	// sort results in decreasing order of count
	sort.Slice(problemCounts, func(i, j int) bool {
		return problemCounts[i].Count > problemCounts[j].Count
	})
	return &rpc.LintStats{ProblemCounts: problemCounts}
}

func mergeLintStats(problemCounts []*rpc.LintProblemCount, lintstats *rpc.LintStats) []*rpc.LintProblemCount {
	for _, pc := range lintstats.ProblemCounts {
		var problemCount *rpc.LintProblemCount
		for _, pc2 := range problemCounts {
			if pc2.RuleId == pc.RuleId {
				problemCount = pc2
				break
			}
		}
		if problemCount == nil {
			problemCount = &rpc.LintProblemCount{
				Count:      0,
				RuleId:     pc.RuleId,
				RuleDocUri: pc.RuleDocUri,
			}
			problemCounts = append(problemCounts, problemCount)
		}
		problemCount.Count += pc.Count
	}
	return problemCounts
}
