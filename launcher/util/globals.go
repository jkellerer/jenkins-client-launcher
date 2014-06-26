// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import "sync"

// Points to the absolute path of the Jenkins client jar.
var ClientJar = ""

// Is updated by the server monitor with the current node state (IDLE means no build is running).
// Note: When server side monitoring is disable the node will ALWAYS appear as idle.
var NodeIsIdle = NewAtomicBoolean()

// Points to the absolute path of the Java executable.
var Java = ""

// Contains an address (HOST:PORT) which mirrors the Jenkins client server port.
var TransportTunnelAddress = ""

// Thread save boolean value.
type AtomicBoolean struct {
	value bool
	mutex sync.Mutex
}

func NewAtomicBoolean() *AtomicBoolean {
	b := new(AtomicBoolean)
	b.mutex = sync.Mutex{}
	return b
}

func (self *AtomicBoolean) Set(value bool) {
	self.mutex.Lock(); defer self.mutex.Unlock()
	self.value = value
}

func (self *AtomicBoolean) Get() bool {
	self.mutex.Lock(); defer self.mutex.Unlock()
	return self.value
}


// Thread save boolean value.
type AtomicInt32 struct {
	value int32
	mutex sync.Mutex
}

func NewAtomicInt32() *AtomicInt32 {
	b := new(AtomicInt32)
	b.mutex = sync.Mutex{}
	return b
}

func (self *AtomicInt32) Set(value int32) {
	self.mutex.Lock(); defer self.mutex.Unlock()
	self.value = value
}

func (self *AtomicInt32) Get() int32 {
	self.mutex.Lock(); defer self.mutex.Unlock()
	return self.value
}
