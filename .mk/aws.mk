# Copyright (c) 2019 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the License);
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an AS IS BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: aws-init
aws-init:
	@pushd scripts/aws
	@./aws-init.sh
	@popd

.PHONY: aws-start
aws-start:
	@pushd scripts/aws
	@AWS_REGION=us-east-2 go run ./... Create
	@popd

.PHONY: aws-restart
aws-restart: aws-destroy aws-start

.PHONY: aws-destroy
aws-destroy:
	@pushd scripts/aws
	@AWS_REGION=us-east-2 go run ./... Delete
	@popd

.PHONY: aws-%-load-images
aws-%-load-images: ;

.PHONY: aws-get-kubeconfig
aws-get-kubeconfig:
	@pushd scripts/aws
	@aws eks update-kubeconfig --name nsm --kubeconfig ../../kubeconfig
	@popd

.PHONY: aws-build
aws-build: $(addsuffix -build,$(addprefix aws-,$(BUILD_CONTAINERS)))

.PHONY: aws-%-build
aws-%-build:
	
	@ssh -i scripts/aws/nsm-key-pair$(NSM_AWS_SERVICE_SUFFIX) 

.PHONY: aws-save
aws-save: $(addsuffix -save,$(addprefix aws-,$(BUILD_CONTAINERS))) ;

.PHONY: aws-%-save
aws-%-save: aws-%-build ;