#!/bin/bash
#
# Copyright 2021 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This script uses the [googleapis](https://github.com/googleapis/googleapis)
# Protocol Buffer descriptions of public Google APIs to build a colllection
# of API descriptions and them performs some analysis on them.
# It assumes that the repo has been cloned to the user's Desktop,
# so that it can be found at `~/Desktop/googleapis`.

# It also assumes an enviroment configured to call a Registry API implementation.
# This includes the registry-server running with a local SQLite database,
# which can be started by running `registry-server -c config/sqlite.yaml`
# from the root of the registry repo. To configure clients to call this
# server, run `source auth/LOCAL.sh` in the shell before running the following
# commands.

# A registry exists under a top-level project.
PROJECT=protos

# First, delete and re-create the project to get a fresh start.
apg admin delete-project --name projects/$PROJECT
apg admin create-project --project_id $PROJECT \
	--project.display_name "Google APIs" \
	--project.description "Protocol buffer descriptions of public Google APIs"

# Get the commit hash of the checked-out protos directory
export COMMIT=`(cd ~/Desktop/googleapis; git rev-parse HEAD)`

# Upload all of the APIs in the googleapis directory at once.
# This happens in parallel and usually takes less than 10 seconds.
registry upload bulk protos \
	--project-id $PROJECT ~/Desktop/googleapis \
	--base-uri https://github.com/googleapis/googleapis/blob/$COMMIT 

# Now compute summary details of all of the APIs in the project. 
# This will log errors if any of the API specs can't be parsed,
# but for every spec that is parsed, this will set the display name
# and description of the corresponding API from the values in the specs.
registry compute details projects/$PROJECT/locations/global/apis/-

# The `registry upload bulk protos` subcommand automatically generated API ids
# from the path to the protos in the repo. List the APIs with the following command:
registry list projects/$PROJECT/locations/global/apis

# We can count them by piping this through `wc -l`.
registry list projects/$PROJECT/locations/global/apis | wc -l

# Many of these APIs have multiple versions. We can list all of the API versions
# by using a "-" wildcard for the API id:
registry list projects/$PROJECT/locations/global/apis/-/versions

# Similarly, we can use wildcards for the version ids and list all of the specs.
# Here you'll see that the spec IDs are "protos.zip". This was set in the registry
# tool, which uploaded each API description as a zip archive of proto files.
registry list projects/$PROJECT/locations/global/apis/-/versions/-/specs

# To see more about an individual spec, use the `registry get` command:
registry get projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip

# You can also get this with the automatically-generated `apg` command line tool:
apg registry get-api-spec --name projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip

# Add the `--json` flag to get this as JSON:
apg registry get-api-spec --name projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip --json

# You might notice that that didn't return the actual spec. That's because the spec contents
# are accessed through a separate method that (when transcoded to HTTP) allows direct download
# of spec contents.
apg registry get-api-spec-contents --name projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip

# An easier way to get the bytes of the spec is to use `registry get` with the `--contents` flag.
# This writes the bytes to stdout, so you probably want to redirect this to a file, as follows:
registry get projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip \
	--contents > protos.zip

# When you unzip this file, you'll find a directory hierarchy suitable for compiling with `protoc`.
# protoc google/cloud/translate/v3/translation_service.proto -o.
# (This requires additional protos that you can find in
# [github.com/googleapis/api-common-protos](https://github.com/googleapis/api-common-protos).

# The registry tool can compute simple complexity metrics for protos stored in the Registry.
registry compute complexity projects/$PROJECT/locations/global/apis/-/versions/-/specs/-

# Complexity results are stored in artifacts associated with the specs.
registry list projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/complexity

# We can use the `registry get` subcommand to read individual complexity records.
registry get projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip/artifacts/complexity

# The registry tool also supports exporting all of the complexity results to a Google sheet.
# (The following command expects OAuth client credentials with access to the
# Google Sheets API to be available locally in ~/.credentials/registry.json)
registry export sheet projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/complexity \
	--as projects/$PROJECT/locations/global/artifacts/complexity-sheet

# We can also compute the vocabulary of proto APIs.
registry compute vocabulary projects/$PROJECT/locations/global/apis/-/versions/-/specs/-

# Vocabularies are also stored as artifacts associated with API specs.
registry get projects/$PROJECT/locations/global/apis/google-cloud-translate/versions/v3/specs/protos.zip/artifacts/vocabulary

# The registry command can perform set operations on vocabularies.
# To find common terms in all Google speech-related APIs, use the following:
registry vocabulary intersection projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/vocabulary \
	--filter "api_id.contains('speech')"

# We can also save this to a property.
registry vocabulary intersection projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/vocabulary \
	--filter "api_id.contains('speech')" --output projects/$PROJECT/locations/global/artifacts/speech-common

# We can then read it directly or export it to a Google Sheet.
registry get projects/$PROJECT/locations/global/artifacts/speech-common
registry export sheet projects/$PROJECT/locations/global/artifacts/speech-common

# To see a larger vocabulary, let's now compute the union of all the vocabularies in our project.
registry vocabulary union projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/vocabulary \
	--output projects/$PROJECT/locations/global/artifacts/vocabulary

# We can also export this with `registry get` but it's easier to view this as a sheet:
registry export sheet projects/$PROJECT/locations/global/artifacts/vocabulary

# You'll notice that usage counts are included for each term, so we can sort by count
# and find the most commonly-used terms across all of our APIs.
# With vocabulary operations we can discover common terms across groups of APIs,
# track changes across versions, and find unique terms in APIs that we are reviewing.
# By storing these results and other artifacts in the Registry, we can build a
# centralized store of API information that can help manage an API program.

# We can also run analysis tools like linters and store the results in the Registry.
# Here we run the Google api-linter and compile summary statistics.
registry compute lint projects/$PROJECT/locations/global/apis/-/versions/-/specs/-
registry compute lintstats projects/$PROJECT/locations/global/apis/-/versions/-/specs/- --linter aip
registry compute lintstats projects/$PROJECT --linter aip


