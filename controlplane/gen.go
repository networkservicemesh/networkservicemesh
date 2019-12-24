package controlplane

//go:generate bash -c "mockgen -destination=./pkg/tests/mock/nsm.mg.go -package=mock -self_package=nsm github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm NetworkServiceEndpointManager,NetworkServiceClient"
