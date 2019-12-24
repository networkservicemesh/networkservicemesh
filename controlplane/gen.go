package controlplane

//go:generate bash -c "mockgen -source=./pkg/api/nsm/nsm.go -destination=./pkg/tests/mock/nsm.mg.go -package=mock"
