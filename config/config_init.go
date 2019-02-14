package config

import (
//"errors"
)

func boolPointer(b bool) *bool {
	return &b
}

// encapsulate logic to make it easier to read in this file
func (config *GudgeonConfig) verifyAndInit() error {
	// collect errors for reporting/combining into one error
	errors := make([]error, 0)

	// network verification
	if config.Network == nil {
		config.Network = &GudgeonNetwork{}
	}
	if err := config.Network.verifyAndInit(); err != nil {
		errors = append(errors, err)
	}

	// web defaults and verification
	if config.Web == nil {
		config.Web = &GudgeonWeb{
			Enabled: true,
			Address: "127.0.0.1",
			Port:    9009,
		}
	}

	// query log configuration
	if config.Metrics == nil {
		config.Metrics = &GudgeonMetrics{}
	}

	if err := config.Metrics.verifyAndInit(); err != nil {
		errors = append(errors, err)
	}

	if config.QueryLog == nil {
		config.QueryLog = &GudgeonQueryLog{}
	}

	if err := config.QueryLog.verifyAndInit(); err != nil {
		errors = append(errors, err)
	}

	return nil
}

func (network *GudgeonNetwork) verifyAndInit() error {
	// set default values for tcp and udp if nil
	if network.TCP == nil {
		network.TCP = boolPointer(true)
	}
	if network.UDP == nil {
		network.UDP = boolPointer(true)
	}
	if network.Systemd == nil {
		network.Systemd = boolPointer(true)
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

func (metrics *GudgeonMetrics) verifyAndInit() error {
	if metrics.Enabled == nil {
		metrics.Enabled = boolPointer(true)
	}

	if "" == metrics.Duration {
		metrics.Duration = "7d"
	}

	if "" == metrics.Interval {
		metrics.Interval = "15s"
	}

	return nil
}

func (ql *GudgeonQueryLog) verifyAndInit() error {
	if ql.Enabled == nil {
		ql.Enabled = boolPointer(true)
	}

	if ql.Stdout == nil {
		ql.Stdout = boolPointer(true)
	}

	if "" == ql.Duration {
		ql.Duration = "7d"
	}

	return nil
}
