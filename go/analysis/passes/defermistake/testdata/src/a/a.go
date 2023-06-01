// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package a

import (
	"fmt"
	"time"
)

func x(time.Duration) {}
func x2(float64)      {}

func _() { // code that doesnt trigger a report.
	// The following are OK because func is not evaluated in defer invocation.
	now := time.Now()
	defer func() {
		fmt.Println(time.Since(now)) // OK because time.Since is not evaluated in defer
	}()
	evalBefore := time.Since(now)
	defer fmt.Println(evalBefore)
	do := func(f func()) {}
	defer do(func() { time.Since(now) })

	// FIXME: The following are okay even though technically time.Since is evaluated here.
	// We don't walk into literal functions.
	defer x((func() time.Duration { return time.Since(now) })())
}

type y struct{}

func (y) A(float64)        {}
func (*y) B(float64)       {}
func (y) C(time.Duration)  {}
func (*y) D(time.Duration) {}

func _() { // code that triggers a report.
	now := time.Now()
	defer time.Since(now)                     // want "defer func should not evaluate time.Since"
	defer fmt.Println(time.Since(now))        // want "defer func should not evaluate time.Since"
	defer fmt.Println(time.Since(time.Now())) // want "defer func should not evaluate time.Since"
	defer x(time.Since(now))                  // want "defer func should not evaluate time.Since"
	defer x2(time.Since(now).Seconds())       // want "defer func should not evaluate time.Since"
	defer y{}.A(time.Since(now).Seconds())    // want "defer func should not evaluate time.Since"
	defer (&y{}).B(time.Since(now).Seconds()) // want "defer func should not evaluate time.Since"
	defer y{}.C(time.Since(now))              // want "defer func should not evaluate time.Since"
	defer (&y{}).D(time.Since(now))           // want "defer func should not evaluate time.Since"
}
