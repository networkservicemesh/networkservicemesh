// Copyright (c) 2019-2020 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package artifacts

import "github.com/networkservicemesh/networkservicemesh/utils"

const (
	printToConsole   utils.EnvVar = "ARTIFACTS_PRINT_INTO_CONSOLE"
	saveAsFiles      utils.EnvVar = "ARTIFACTS_SAVE_FILES"
	saveInAnyCase    utils.EnvVar = "ARTIFACTS_SAVE_ALWAYS"
	archiveArtifacts utils.EnvVar = "ARTIFACTS_ARCHIVE"
	outputDirectory  utils.EnvVar = "ARTIFACTS_DIR"
)

//NeedToSave returns true if any of saving artifact option envs is not empty.
func NeedToSave() bool {
	return saveAsFiles.GetBooleanOrDefault(false) ||
		archiveArtifacts.GetBooleanOrDefault(false)
}
