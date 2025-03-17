// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package process

import (
	"fmt"
	"os/exec"
	"test-runner/util"
)

const (
	cloudWatchAgentCtl = "/opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl"
	adotCtl            = "/opt/aws/aws-otel-collector/bin/aws-otel-collector-ctl"

	cloudWatchAgentStateDir = "/opt/aws/amazon-cloudwatch-agent/logs/state/"
	fluentBitStateDir       = "/var/fluent-bit/state"
)

type Manager struct {
	Start     Func
	Teardown  Func
	Configure func([]string) error
	Stop      Func
}

func (m Manager) Reconfigure(configs []string) error {
	if err := m.Stop(); err != nil {
		return err
	}
	if err := m.Teardown(); err != nil {
		return err
	}
	if err := m.Configure(configs); err != nil {
		return err
	}
	return m.Start()
}

type Func func() error

var (
	managers = map[Process]Manager{
		CloudWatchAgent: {
			Start: execCommandFn(cloudWatchAgentCtl, "-a", "start"),
			Stop:  execCommandFn(cloudWatchAgentCtl, "-a", "stop"),
			Teardown: func() error {
				return util.ClearDir(cloudWatchAgentStateDir)
			},
			Configure: func(configs []string) error {
				if len(configs) >= 1 {
					fn := execCommandFn(cloudWatchAgentCtl, "-a", "fetch-config", "-c", fmt.Sprintf("file:%s", configs[0]))
					if err := fn(); err != nil {
						return err
					}
				}
				if len(configs) >= 2 {
					for _, config := range configs[1:] {
						fn := execCommandFn(cloudWatchAgentCtl, "-a", "append-config", "-c", fmt.Sprintf("file:%s", config))
						if err := fn(); err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
		FluentBit: {
			Start: execCommandFn("systemctl", "start", "fluent-bit"),
			Stop:  execCommandFn("systemctl", "stop", "fluent-bit"),
			Teardown: func() error {
				return util.ClearDir(fluentBitStateDir)
			},
			Configure: func(configs []string) error {
				if len(configs) != 1 {
					return fmt.Errorf("expected one fluent-bit config, got %d", len(configs))
				}
				fn := execCommandFn("cp", configs[0], "/etc/fluent-bit/fluent-bit.conf")
				if err := fn(); err != nil {
					return err
				}
				return nil
			},
		},
		ADOT: {
			// started during configure
			Start: func() error {
				return nil
			},
			Stop: execCommandFn(adotCtl, "-a", "stop"),
			// no state
			Teardown: func() error {
				return nil
			},
			Configure: func(configs []string) error {
				if len(configs) != 1 {
					return fmt.Errorf("expected one aws-otel-collector config, got %d", len(configs))
				}
				fn := execCommandFn(adotCtl, "-a", "start", "-c", fmt.Sprintf("file:%s", configs[0]))
				if err := fn(); err != nil {
					return err
				}
				return nil
			},
		},
	}
)

func execCommandFn(name string, args ...string) Func {
	return func() error {
		cmd := exec.Command(name, args...)
		return cmd.Run()
	}
}
