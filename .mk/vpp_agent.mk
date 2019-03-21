ARCH ?= $(shell uname -m)
ifeq (${ARCH}, x86_64)
  export VPP_AGENT=ligato/vpp-agent:latest
  export VPP_AGENT_DEV=ligato/dev-vpp-agent:latest
endif
ifeq (${ARCH}, aarch64)
  export VPP_AGENT=ligato/vpp-agent-arm64:latest
  export VPP_AGENT_DEV=ligato/dev-vpp-agent-arm64:latest
endif
