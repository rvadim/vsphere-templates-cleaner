package main

import (
	"context"
	"github.com/vmware/govmomi/object"
	"regexp"
	"testing"
)

type VMMock struct {
	name string
}

func (vm *VMMock) Name() string {
	return vm.name
}

func (vm *VMMock) Destroy(ctx context.Context) (*object.Task, error) {
	return nil, nil
}

func TestGetTemplate(t *testing.T) {
	vm1 := &VMMock{name: "ubuntu-16.04-base-123"}
	vm2 := &VMMock{name: "ubuntu-16.04-123"}
	re := regexp.MustCompile("ubuntu-16.04-base-([0-9]+)")
	output := getTemplate(re, vm1)

	if output.version != 123 {
		t.Errorf("Output version must be 123, but actual %d", output.version)
	}

	if output.name != "ubuntu-16.04-base-123" {
		t.Errorf("Output name must be 'ubuntu-16.04-base-123', but actual %s", output.name)
	}

	if getTemplate(re, vm2) != nil {
		t.Errorf("Output must be nil, but %v", output)
	}

	vm3 := &VMMock{name: ""}
	re = regexp.MustCompile("^(.*)-([0-9]+)$")
	output = getTemplate(re, vm3)
	if output != nil {
		t.Errorf("Output must be nil, but %v", output)
	}
}