ifeq (${BUILD_ARCH}, amd64)
  export VPP_AGENT=ligato/vpp-agent:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:v2.1.1
endif
ifeq (${BUILD_ARCH}, arm64)
  export VPP_AGENT=ligato/vpp-agent-arm64:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:v2.1.1
endif

.PHONY: docker-vppagent-dataplane-dev-build
docker-vppagent-dataplane-dev-build: docker-vppagent-dataplane-build
	@${DOCKERBUILD} --network="host" --build-arg VPP_AGENT=${VPP_AGENT} --build-arg VPP_DEV=${VPP_AGENT_DEV} --build-arg REPO=${ORG}  --build-arg VERSION=${VERSION}  -t ${ORG}/vppagent-dataplane-dev -f docker/Dockerfile.vppagent-dataplane-dev . && \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/vppagent-dataplane-dev ${ORG}/vppagent-dataplane-dev:${COMMIT} ;\
	fi

.PHONY: docker-vppagent-dataplane-dev-build-arm64
docker-vppagent-dataplane-dev-build-arm64: docker-vppagent-dataplane-build-arm64
	@${DOCKERBUILD} --network="host" --build-arg VPP_AGENT=${VPP_AGENT} --build-arg VPP_DEV=${VPP_AGENT_DEV} --build-arg REPO=${ORG} -t ${ORG}/vppagent-dataplane-dev-arm64 -f docker/Dockerfile.vppagent-dataplane-dev-arm64 . && \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/vppagent-dataplane-dev-arm64 ${ORG}/vppagent-dataplane-dev-arm64:${COMMIT} ;\
	fi