/*
Copyright 2022 The Koordinator Authors.

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

package resourceexecutor

import (
	"errors"
	"fmt"

	sysutil "github.com/clay-wangzhi/koordinator/pkg/koordlet/util/system"
)

var ErrResourceNotRegistered = errors.New("resource not registered")

type CgroupReader interface {
	ReadCPUQuota(parentDir string) (int64, error)
	ReadCPUPeriod(parentDir string) (int64, error)
	ReadCPUShares(parentDir string) (int64, error)
	ReadCPUStat(parentDir string) (*sysutil.CPUStatRaw, error)
	ReadCPUProcs(parentDir string) ([]uint32, error)
}

var _ CgroupReader = &CgroupV1Reader{}

type CgroupV1Reader struct{}

func (r *CgroupV1Reader) ReadCPUQuota(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV1, sysutil.CPUCFSQuotaName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}
	return readCgroupAndParseInt64(parentDir, resource)
}

func (r *CgroupV1Reader) ReadCPUPeriod(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV1, sysutil.CPUCFSPeriodName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}
	return readCgroupAndParseInt64(parentDir, resource)
}

func (r *CgroupV1Reader) ReadCPUShares(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV1, sysutil.CPUSharesName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}
	return readCgroupAndParseInt64(parentDir, resource)
}

func (r *CgroupV1Reader) ReadCPUStat(parentDir string) (*sysutil.CPUStatRaw, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV1, sysutil.CPUStatName)
	if !ok {
		return nil, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil {
		return nil, err
	}
	// content: "nr_periods 0\nnr_throttled 0\nthrottled_time 0\n..."
	v, err := sysutil.ParseCPUStatRaw(s)
	if err != nil {
		return nil, fmt.Errorf("cannot parse cgroup value %s, err: %v", s, err)
	}
	return v, nil
}

func (r *CgroupV1Reader) ReadCPUProcs(parentDir string) ([]uint32, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV1, sysutil.CPUProcsName)
	if !ok {
		return nil, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil {
		return nil, err
	}

	// content: `7742\n10971\n11049\n11051...`
	return sysutil.ParseCgroupProcs(s)
}

var _ CgroupReader = &CgroupV2Reader{}

type CgroupV2Reader struct{}

func (r *CgroupV2Reader) ReadCPUQuota(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV2, sysutil.CPUCFSQuotaName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil && IsCgroupDirErr(err) {
		return -1, err
	} else if err != nil {
		return -1, fmt.Errorf("cannot read cgroup file, err: %v", err)
	}

	// content: "max 100000", "100000 100000"
	v, err := sysutil.ParseCPUCFSQuotaV2(s)
	if err != nil {
		return -1, fmt.Errorf("cannot parse cgroup value %s, err: %v", s, err)
	}
	return v, nil
}

func (r *CgroupV2Reader) ReadCPUPeriod(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV2, sysutil.CPUCFSPeriodName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil {
		return -1, err
	}

	// content: "max 100000", "100000 100000"
	v, err := sysutil.ParseCPUCFSPeriodV2(s)
	if err != nil {
		return -1, fmt.Errorf("cannot parse cgroup value %s, err: %v", s, err)
	}
	return v, nil
}

func (r *CgroupV2Reader) ReadCPUShares(parentDir string) (int64, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV2, sysutil.CPUSharesName)
	if !ok {
		return -1, ErrResourceNotRegistered
	}

	v, err := readCgroupAndParseInt64(parentDir, resource)
	if err != nil {
		return -1, err
	}
	// convert cpu.weight value into cpu.shares value
	return sysutil.ConvertCPUWeightToShares(v)
}

func (r *CgroupV2Reader) ReadCPUStat(parentDir string) (*sysutil.CPUStatRaw, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV2, sysutil.CPUStatName)
	if !ok {
		return nil, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil {
		return nil, err
	}
	// content: "...\nnr_periods 0\nnr_throttled 0\nthrottled_usec 0\n..."
	v, err := sysutil.ParseCPUStatRawV2(s)
	if err != nil {
		return nil, fmt.Errorf("cannot parse cgroup value %s, err: %v", s, err)
	}
	return v, nil
}

func (r *CgroupV2Reader) ReadCPUProcs(parentDir string) ([]uint32, error) {
	resource, ok := sysutil.DefaultRegistry.Get(sysutil.CgroupVersionV2, sysutil.CPUProcsName)
	if !ok {
		return nil, ErrResourceNotRegistered
	}
	s, err := cgroupFileRead(parentDir, resource)
	if err != nil {
		return nil, err
	}

	// content: `7742\n10971\n11049\n11051...`
	return sysutil.ParseCgroupProcs(s)
}

func NewCgroupReader() CgroupReader {
	if sysutil.GetCurrentCgroupVersion() == sysutil.CgroupVersionV2 {
		return &CgroupV2Reader{}
	}
	return &CgroupV1Reader{}
}
