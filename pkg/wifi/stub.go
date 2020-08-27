// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wifi

import "io"

var _ = WiFi(&StubWorker{})

type StubWorker struct {
	Options []Option
	ID      string
}

func NewStubWorker(stdout, stderr io.Writer, id string, options ...Option) (WiFi, error) {
	return &StubWorker{ID: id, Options: options}, nil
}

func (w *StubWorker) Scan(stdout, stderr io.Writer) ([]Option, error) {
	return w.Options, nil
}

func (w *StubWorker) GetID(stdout, stderr io.Writer) (string, error) {
	return w.ID, nil
}

func (*StubWorker) Connect(stdout, stderr io.Writer, a ...string) error {
	return nil
}
