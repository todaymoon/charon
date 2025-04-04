// Copyright © 2022-2025 Obol Labs Inc. Licensed under the terms of a Business Source License 1.1

// Code generated by mockery v2.42.1. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/obolnetwork/charon/core"
	mock "github.com/stretchr/testify/mock"

	protocol "github.com/libp2p/go-libp2p/core/protocol"
)

// Consensus is an autogenerated mock type for the Consensus type
type Consensus struct {
	mock.Mock
}

// Participate provides a mock function with given fields: _a0, _a1
func (_m *Consensus) Participate(_a0 context.Context, _a1 core.Duty) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Participate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, core.Duty) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Propose provides a mock function with given fields: _a0, _a1, _a2
func (_m *Consensus) Propose(_a0 context.Context, _a1 core.Duty, _a2 core.UnsignedDataSet) error {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for Propose")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, core.Duty, core.UnsignedDataSet) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ProtocolID provides a mock function with given fields:
func (_m *Consensus) ProtocolID() protocol.ID {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ProtocolID")
	}

	var r0 protocol.ID
	if rf, ok := ret.Get(0).(func() protocol.ID); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(protocol.ID)
	}

	return r0
}

// Start provides a mock function with given fields: _a0
func (_m *Consensus) Start(_a0 context.Context) {
	_m.Called(_a0)
}

// Subscribe provides a mock function with given fields: _a0
func (_m *Consensus) Subscribe(_a0 func(context.Context, core.Duty, core.UnsignedDataSet) error) {
	_m.Called(_a0)
}

// NewConsensus creates a new instance of Consensus. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewConsensus(t interface {
	mock.TestingT
	Cleanup(func())
}) *Consensus {
	mock := &Consensus{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
