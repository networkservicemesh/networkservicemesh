package serviceregistry

//go:generate bash -c "mockgen -destination=../tests/mock/serviceregistry.mg.go -package=tests -self_package=serviceregistry github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry ServiceRegistry"
