# non-cross platform build:
export IMAGE_ARCH=""
export GOARCH=""
export DOCKERARCHI=""
is_cross_build=fake-qemu #  workaround to overcome non-conditional Dockerfile COPY clause

# the first supported target platform is amd64
ifeq (${ARCH}, amd64)
  export VPP_AGENT=ligato/vpp-agent:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:v2.1.1
  export IMAGE_ARCH=""
  #cross platform build on arm64 is not supported
  #ifeq (${OS_ARCH}, aarch64)
  #  export GOARCH="GOARCH=amd64"
  #  is_cross_build=install-qemu
  #endif
endif

# the second supported target platform is arm64
ifeq (${ARCH}, arm64)
  export VPP_AGENT=ligato/vpp-agent-arm64:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:v2.1.1
  export IMAGE_ARCH="-arm64"
  #cross platform build on x86_64 platform is supported:
  ifeq (${OS_ARCH}, x86_64)
    export GOARCH="arm64"
    export DOCKERARCHI="arm64v8/"
    is_cross_build=install-qemu
  endif
endif

.PHONY: docker-vppagent-dataplane-dev-build
docker-vppagent-dataplane-dev-build: $(is_cross_build) docker-vppagent-dataplane-build
	@${DOCKERBUILD} --network="host" \
	                --build-arg GOARCH=${GOARCH} \
	                --build-arg VPP_AGENT=${VPP_AGENT} \
	                --build-arg VPP_DEV=${VPP_AGENT_DEV} \
	                --build-arg REPO=${ORG} \
	                --build-arg VERSION=${VERSION} \
	                -t ${ORG}/vppagent-dataplane-dev${IMAGE_ARCH} \
	                -f docker/Dockerfile.vppagent-dataplane-dev . && \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/vppagent-dataplane-dev ${ORG}/vppagent-dataplane-dev${IMAGE_ARCH}:${COMMIT} ;\
	fi

.PHONY: install-qemu
install-qemu: fake-qemu
	# prepare for arm64 docker image built on amd64 platform
	# Note that currently this is executed only in case OS_ARCH=x86_64 and ARCH=arm64 because in Makefile are restricted supported combinations of OS_ARCH and ARCH
	case "${OS_ARCH}" in
	  x86_64)
	    cp /usr/bin/qemu-aarch64-static ${REPO_FOLDER}/test/applications/build/qemu-aarch64-static
	    cp /usr/bin/qemu-aarch64-static ${REPO_FOLDER}/dataplane/vppagent/conf/vpp/qemu-aarch64-static
	    cp /usr/bin/qemu-aarch64-static ${REPO_FOLDER}/test/applications/build/qemu-aarch64-static
	  #arm64)
	  #  cp /usr/bin/qemu-x86_64-static ${REPO_FOLDER}/test/applications/build/qemu-x86_64-static
	  #  cp /usr/bin/qemu-x86_64-static ${REPO_FOLDER}/dataplane/vppagent/conf/vpp/qemu-x86_64-static
	esac

.PHONY: fake-qemu
fake-qemu:
	# prepare for amd64 target built on amd64 platform or for arm64 target built on arm64 platform
	touch ${REPO_FOLDER}/qemu-FAKE-static
	echo 'echo "This file is here because Dockerfile clause COPY could not work conditionally."' > ${REPO_FOLDER}/qemu-FAKE-static
	chmod +x ${REPO_FOLDER}/qemu-FAKE-static
	cp ${REPO_FOLDER}/qemu-FAKE-static ${REPO_FOLDER}/test/applications/build/qemu-aarch64-static
	cp ${REPO_FOLDER}/qemu-FAKE-static ${REPO_FOLDER}/dataplane/vppagent/conf/vpp/qemu-aarch64-static
	cp ${REPO_FOLDER}/qemu-FAKE-static ${REPO_FOLDER}/test/applications/build/qemu-aarch64-static
	#cp ${REPO_FOLDER}/qemu-FAKE-static ${REPO_FOLDER}/test/applications/build/qemu-x86_64-static
	#cp ${REPO_FOLDER}/qemu-FAKE-static ${REPO_FOLDER}/dataplane/vppagent/conf/vpp/qemu-x86_64-static
