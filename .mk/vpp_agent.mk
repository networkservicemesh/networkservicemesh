ARCH ?= $(shell uname -m)
ifeq (${ARCH}, x86_64)
  export VPP_AGENT=ligato/vpp-agent:v1.8.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:v1.8.1
endif
ifeq (${ARCH}, aarch64)
  export VPP_AGENT=ligato/vpp-agent-arm64:v1.8.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:v1.8.1
endif
