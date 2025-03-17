// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package process

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Process int

const (
	CloudWatchAgent Process = iota
	FluentBit
	ADOT
)

var _ yaml.Unmarshaler = (*Process)(nil)
var _ yaml.Marshaler = (*Process)(nil)

func (p Process) String() string {
	return []string{
		"amazon-cloudwatch-agent",
		"fluent-bit",
		"aws-otel-collector",
	}[p]
}

func (p Process) Manager() (*Manager, error) {
	if manager, ok := managers[p]; ok {
		return &manager, nil
	}
	return nil, fmt.Errorf("no manager found for process type: %s", p)
}

func (p *Process) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected scalar node")
	}
	processStr := strings.ToLower(value.Value)
	switch processStr {
	case CloudWatchAgent.String():
		*p = CloudWatchAgent
	case FluentBit.String():
		*p = FluentBit
	case ADOT.String():
		*p = ADOT
	default:
		return fmt.Errorf("unknown process type: %s", processStr)
	}
	return nil
}

func (p Process) MarshalYAML() (interface{}, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: p.String(),
	}, nil
}
