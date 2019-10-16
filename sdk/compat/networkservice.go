package compat

import (
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
)

func NetworkServiceRequestUnifiedToLocal(r *unified.NetworkServiceRequest) *local.NetworkServiceRequest {
	if r == nil {
		return nil
	}
	return &local.NetworkServiceRequest{
		Connection:           ConnectionUnifiedToLocal(r.GetConnection()),
		MechanismPreferences: MechanismListUnifiedToLocal(r.GetMechanismPreferences()),
	}
}

func NetworkServiceRequestLocalToUnified(r *local.NetworkServiceRequest) *unified.NetworkServiceRequest {
	if r == nil {
		return nil
	}
	return &unified.NetworkServiceRequest{
		Connection:           ConnectionLocalToUnified(r.GetConnection()),
		MechanismPreferences: MechanismListLocalToUnified(r.GetMechanismPreferences()),
	}
}

func NetworkServiceRequestUnifiedToRemote(r *unified.NetworkServiceRequest) *remote.NetworkServiceRequest {
	if r == nil {
		return nil
	}
	return &remote.NetworkServiceRequest{
		Connection:           ConnectionUnifiedToRemote(r.GetConnection()),
		MechanismPreferences: MechanismListUnifiedToRemote(r.GetMechanismPreferences()),
	}
}

func NetworkServiceRequestRemoteToUnified(r *remote.NetworkServiceRequest) *unified.NetworkServiceRequest {
	if r == nil {
		return nil
	}
	return &unified.NetworkServiceRequest{
		Connection:           ConnectionRemoteToUnified(r.GetConnection()),
		MechanismPreferences: MechanismListRemoteToUnified(r.GetMechanismPreferences()),
	}
}
