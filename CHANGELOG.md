## NSM Release 0.1.0 (Unreleased)

FEATURES:

- spec: Mutating Admission Controller injecting NSM init container into pod spec - #708
- spec: Readiness probes for nsmd, vppagent-forwarder - #711
- core: Add liveness/readiness probes for nsmd and vppagent-forwarder #730

IMPROVEMENTS:

- core: Implement NSMD check for NSE troubles and initial Auto healing operation - #647
- general: Update proto generation for go modules - #749
- core: Replace nsm.ligato.io/socket - #755
- core: Fix nsmd containers handlings of signals so they can terminate fast - #746
- general: Split the repo to core NSM and examples - #610
- general: Add PULL_REQUEST_TEMPLATE.md - #705

BUG FIXES:

- core: TestNSCAndICMPRemote is failed because of VPP forwarder error - #725
- core: Add check of cmdline during resolving namespace by inode - #748
