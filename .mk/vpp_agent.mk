ARCH ?= $(shell uname -m)
ifeq (${ARCH}, x86_64)
  export VPP_AGENT=ligato/vpp-agent:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:v2.1.1
endif
ifeq (${ARCH}, aarch64)
  export VPP_AGENT=ligato/vpp-agent-arm64:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:v2.1.1
endif

.PHONY: docker-vppagent-dataplane-dev-build
docker-vppagent-dataplane-dev-build: docker-vppagent-dataplane-build
	@${DOCKERBUILD} --network="host" --build-arg GO_VERSION=${GO_VERSION} --build-arg VPP_AGENT=${VPP_AGENT} --build-arg VENDORING="${VENDORING}" --build-arg VPP_DEV=${VPP_AGENT_DEV} --build-arg REPO=${ORG}  --build-arg VERSION=${VERSION}  -t ${ORG}/vppagent-dataplane-dev -f docker/Dockerfile.vppagent-dataplane-dev . && \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/vppagent-dataplane-dev ${ORG}/vppagent-dataplane-dev:${COMMIT} ;\
	fi