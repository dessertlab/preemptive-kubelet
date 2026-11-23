/*
Copyright 2026 Università degli Studi di Napoli Federico II.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package rtsched runs code under the SCHED_FIFO scheduling policy.
package rtsched

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

// schedParam mirrors the C struct sched_param used by sched_setscheduler.
type schedParam struct {
	Sched_priority int32
}

// RunWithFIFO runs fn on an OS thread locked for the duration of fn, under
// SCHED_FIFO with the given priority, and restores SCHED_OTHER afterwards.
// If the policy cannot be set, fn is run with the default policy.
func RunWithFIFO(priority int32, fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tid := unix.Gettid()
	if err := setScheduler(tid, unix.SCHED_FIFO, priority); err != nil {
		klog.ErrorS(err, "Failed to set SCHED_FIFO", "tid", tid, "priority", priority)
		fn()
		return
	}
	defer func() {
		if err := setScheduler(tid, unix.SCHED_NORMAL, 0); err != nil {
			klog.ErrorS(err, "Failed to restore SCHED_OTHER", "tid", tid)
		}
	}()

	fn()
}

// setScheduler issues the sched_setscheduler syscall for thread tid with the
// given policy and priority.
func setScheduler(tid, policy int, priority int32) error {
	param := schedParam{Sched_priority: priority}
	if _, _, errno := unix.Syscall(
		unix.SYS_SCHED_SETSCHEDULER,
		uintptr(tid),
		uintptr(policy),
		uintptr(unsafe.Pointer(&param)),
	); errno != 0 {
		return fmt.Errorf("sched_setscheduler(tid=%d, policy=%d, priority=%d) failed: %w", tid, policy, priority, errno)
	}
	return nil
}
