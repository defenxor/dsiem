// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build ignore

package darwin

/*
#include <libproc.h>
#include <mach/mach_host.h>
*/
import "C"

type processState uint32

const (
	stateSIDL processState = iota + 1
	stateRun
	stateSleep
	stateStop
	stateZombie
)

const argMax = C.ARG_MAX

type bsdInfo C.struct_proc_bsdinfo

type procTaskInfo C.struct_proc_taskinfo

type procTaskAllInfo C.struct_proc_taskallinfo

type vinfoStat C.struct_vinfo_stat

type fsid C.struct_fsid

type vnodeInfo C.struct_vnode_info

type vnodeInfoPath C.struct_vnode_info_path

type procVnodePathInfo C.struct_proc_vnodepathinfo

type vmStatisticsData C.vm_statistics_data_t

type vmStatistics64Data C.vm_statistics64_data_t

type vmSize C.vm_size_t

const (
	cpuStateUser   = C.CPU_STATE_USER
	cpuStateSystem = C.CPU_STATE_SYSTEM
	cpuStateIdle   = C.CPU_STATE_IDLE
	cpuStateNice   = C.CPU_STATE_NICE
)

type hostCPULoadInfo C.host_cpu_load_info_data_t
