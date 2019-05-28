ARCH ?= $(shell uname -m)
ifeq (${ARCH}, x86_64)
  export VPP_AGENT=ligato/vpp-agent:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:v2.1.1
endif
ifeq (${ARCH}, aarch64)
  export VPP_AGENT=ligato/vpp-agent-arm64:v2.1.1
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:v2.1.1
endif
