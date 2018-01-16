// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !noexec

package collector

import (
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

/*
#include <sys/param.h>
#include <sys/vmmeter.h>
#include <uvm/uvmexp.h>
*/
import "C"

type execCollector struct {
	intr         *prometheus.Desc
	ctxt         *prometheus.Desc
	forks        *prometheus.Desc
	btime        *prometheus.Desc
	procsRunning *prometheus.Desc
	procsBlocked *prometheus.Desc
}

func init() {
	registerCollector("exec", defaultEnabled, NewExecCollector)
}

func NewExecCollector() (Collector, error) {
	return &execCollector{
		intr: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "intr"),
			"Total number of interrupts serviced.",
			nil, nil,
		),
		ctxt: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "context_switches"),
			"Total number of context switches.",
			nil, nil,
		),
		forks: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "forks"),
			"Total number of forks.",
			nil, nil,
		),
		btime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "boot_time"),
			"Node boot time, in unixtime.",
			nil, nil,
		),
		procsRunning: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "procs_running"),
			"Number of processes in runnable state.",
			nil, nil,
		),
		procsBlocked: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "procs_blocked"),
			"Number of processes blocked waiting for I/O to complete.",
			nil, nil,
		),
	}, nil
}

func (c *execCollector) Update(ch chan<- prometheus.Metric) error {
	nintr, err := unix.SysctlUint32("kern.intrcnt.nintrcnt")
	intrsum := 0
	if err != nil {
		return err
	}
	for i := 0; i < int(nintr); i++ {
		// TODO: query KERN_INTRCNT_NUM
		intrcnt := 0
		intrsum += intrcnt
	}
	ch <- prometheus.MustNewConstMetric(c.intr, prometheus.CounterValue, float64(intrsum))

	uvmexpb, err := unix.SysctlRaw("vm.uvmexp")
	if err != nil {
		return err
	}
	uvmexp := *(*C.struct_uvmexp)(unsafe.Pointer(&uvmexpb[0]))
	ch <- prometheus.MustNewConstMetric(c.forks, prometheus.CounterValue, float64(uvmexp.forks))
	ch <- prometheus.MustNewConstMetric(c.ctxt, prometheus.CounterValue, float64(uvmexp.swtch))

	bootb, err := unix.SysctlRaw("kern.boottime")
	if err != nil {
		return err
	}
	boot := *(*unix.Timeval)(unsafe.Pointer(&bootb[0]))
	ch <- prometheus.MustNewConstMetric(c.btime, prometheus.GaugeValue, float64(boot.Sec))

	vmtotalb, err := unix.SysctlRaw("vm.vmmeter")
	if err != nil {
		return err
	}
	vmtotal := *(*C.struct_vmtotal)(unsafe.Pointer(&vmtotalb[0]))
	ch <- prometheus.MustNewConstMetric(c.procsRunning, prometheus.GaugeValue,
		float64(vmtotal.t_rq-1))
	ch <- prometheus.MustNewConstMetric(c.procsBlocked, prometheus.GaugeValue,
		float64(vmtotal.t_dw+vmtotal.t_pw))

	return nil
}
