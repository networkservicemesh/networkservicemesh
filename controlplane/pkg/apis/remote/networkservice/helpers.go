package networkservice

import "fmt"

func (ns *NetworkServiceRequest) IsValid() error {
	if ns == nil {
		return fmt.Errorf("NetorkServiceRequest cannot be nil")
	}

	if ns.GetConnection() == nil {
		return fmt.Errorf("NetworkServiceRequest.Connection cannot be nil %v", ns)
	}

	if err := ns.GetConnection().IsValid(); err != nil {
		return fmt.Errorf("NetworkServiceRequest.Connection is invalid: %s: %v", err, ns)
	}

	if ns.GetMechanismPreferences() == nil {
		return fmt.Errorf("NetworkServiceRequest.MechanismPreferences cannot be nil: %v", ns)
	}
	if len(ns.GetMechanismPreferences()) < 1 {
		return fmt.Errorf("NetworkServiceRequest.MechanismPreferences must have at least one entry: %v", ns)
	}
	return nil
}
