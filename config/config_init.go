package config

// encapsulate logic to make it easier to read in this file
func (config *GudgeonConfig) verifyAndInit() error {

	// network verification
	if config.Network != nil {
		err := config.Network.verifyAndInit()
		if err != nil {
			return err
		}
	} else {
		config.Network = &GudgeonNetwork{}
	}

	return nil
}

func (network *GudgeonNetwork) verifyAndInit() error {
	// set default values for tcp and udp if nil
	if network.TCP == nil {
		network.TCP = func(b bool) *bool { return &b }(true)
	}
	if network.UDP == nil {
		network.UDP = func(b bool) *bool { return &b }(true)
	}

	// do the same for all configured interfaces
	for _, iface := range network.Interfaces {
		if iface.TCP == nil {
			iface.TCP = network.TCP
		}
		if iface.UDP == nil {
			iface.UDP = network.UDP
		}
	}

	return nil
}
