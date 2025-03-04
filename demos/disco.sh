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

# This script uses the [Google API Discovery Service](https://developers.google.com/discovery)
# to build a collection of Google API descriptions and then performs some analysis on these
# discovery documents. Because it calls the Discovery Service directly, no files need to be
# downloaded to the user's local system.

# The script assumes an environment configured to call a Registry API implementation.
# This includes the registry-server running with a local SQLite database,
# which can be started by running `registry-server -c config/sqlite.yaml`
# from the root of the registry repo. To configure clients to call this
# server, run `source auth/LOCAL.sh` in the shell before running the following
# commands.

# A registry exists under a top-level project.
PROJECT=disco

# First, delete and re-create the "disco" project to get a fresh start.
apg admin delete-project --name projects/$PROJECT
apg admin create-project --project_id $PROJECT \
	--project.display_name "Discovery" \
	--project.description "Descriptions of public Google APIs from the API Discovery Service"

# Upload all of the APIs from the Discovery Service at once.
# This happens in parallel and usually takes a minute or two.
registry upload bulk discovery \
	--project-id $PROJECT

# Now compute summary details of all of the APIs in the project. 
# This will log errors if any of the API specs can't be parsed,
# but for every spec that is parsed, this will set the display name
# and description of the corresponding API from the values in the specs.
registry compute details projects/$PROJECT/locations/global/apis/-

# We can list the APIs with the following command:
registry list projects/$PROJECT/locations/global/apis

# We can count the APIs by piping this through `wc -l`.
registry list projects/$PROJECT/locations/global/apis | wc -l

# Many of these APIs have multiple versions. We can list all of the API versions
# by using a "-" wildcard for the API id:
registry list projects/$PROJECT/locations/global/apis/-/versions

# Similarly, we can use wildcards for the version ids and list all of the specs.
# Here you'll see that the spec IDs are "discovery.json". This was set in the registry
# tool, which uploaded each API description as gzipped JSON.
registry list projects/$PROJECT/locations/global/apis/-/versions/-/specs

# To see more about an individual spec, use the `registry get` command:
registry get projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json

# You can also get this with the automatically-generated `apg` command line tool:
apg registry get-api-spec \
	--name projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json

# Add the `--json` flag to get this as JSON:
apg registry get-api-spec --json \
	--name projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json

# You might notice that that didn't return the actual spec. That's because the spec contents
# are accessed through a separate method that (when transcoded to HTTP) allows direct download
# of spec contents.
apg registry get-api-spec-contents \
	--name projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json

# Another way to get the bytes of the spec is to use `registry get` with the `--contents` flag.
registry get projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json \
	--contents

# The registry tool can compute simple complexity metrics for Discovery documents stored in the Registry.
registry compute complexity projects/$PROJECT/locations/global/apis/-/versions/-/specs/-

# Complexity results are stored in artifacts associated with the specs.
registry list projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/complexity

# We can use the `registry get` subcommand to read individual complexity records.
registry get projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json/artifacts/complexity

# The registry tool also supports exporting all of the complexity results to a Google sheet.
# (The following command expects OAuth client credentials with access to the
# Google Sheets API to be available locally in ~/.credentials/registry.json)
registry export sheet projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/complexity \
	--as projects/$PROJECT/locations/global/artifacts/complexity-sheet

# We can also compute the vocabulary of APIs described with Discovery documents.
registry compute vocabulary projects/$PROJECT/locations/global/apis/-/versions/-/specs/-

# Vocabularies are also stored as artifacts associated with API specs.
registry get projects/$PROJECT/locations/global/apis/translate/versions/v3/specs/discovery.json/artifacts/vocabulary

# The registry command can perform set operations on vocabularies.
# To find common terms in all Google speech-related APIs, use the following:
registry vocabulary intersection projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/vocabulary \
	--filter "api_id.contains('speech')"

# We can also save this to a property.
registry vocabulary intersection projects/$PROJECT/locations/global/apis/-/versions/-/specs/-/artifacts/vocabulary \
	--filter "api_id.contains('speech')" --output projects/$PROJECT/locations/global/artifacts/speech-common

# We can then read it directly or export it to a Google Sheet.
registry get projects/$PROJECT/locations/global/artifacts/speech-common --contents
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
