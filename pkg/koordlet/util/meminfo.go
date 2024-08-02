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

package util

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/util/system"
)

const (
	Hugepage1Gkbyte = 1048576
	Hugepage2Mkbyte = 2048
)

// MemInfo is the content of system /proc/meminfo.
// NOTE: the unit of each field is KiB.
type MemInfo struct {
	MemTotal          uint64 `json:"mem_total"`
	MemFree           uint64 `json:"mem_free"`
	MemAvailable      uint64 `json:"mem_available"`
	Buffers           uint64 `json:"buffers"`
	Cached            uint64 `json:"cached"`
	SwapCached        uint64 `json:"swap_cached"`
	Active            uint64 `json:"active"`
	Inactive          uint64 `json:"inactive"`
	ActiveAnon        uint64 `json:"active_anon" field:"Active(anon)"`
	InactiveAnon      uint64 `json:"inactive_anon" field:"Inactive(anon)"`
	ActiveFile        uint64 `json:"active_file" field:"Active(file)"`
	InactiveFile      uint64 `json:"inactive_file" field:"Inactive(file)"`
	Unevictable       uint64 `json:"unevictable"`
	Mlocked           uint64 `json:"mlocked"`
	SwapTotal         uint64 `json:"swap_total"`
	SwapFree          uint64 `json:"swap_free"`
	Dirty             uint64 `json:"dirty"`
	Writeback         uint64 `json:"write_back"`
	AnonPages         uint64 `json:"anon_pages"`
	Mapped            uint64 `json:"mapped"`
	Shmem             uint64 `json:"shmem"`
	Slab              uint64 `json:"slab"`
	SReclaimable      uint64 `json:"s_reclaimable"`
	SUnreclaim        uint64 `json:"s_unclaim"`
	KernelStack       uint64 `json:"kernel_stack"`
	PageTables        uint64 `json:"page_tables"`
	NFS_Unstable      uint64 `json:"nfs_unstable"`
	Bounce            uint64 `json:"bounce"`
	WritebackTmp      uint64 `json:"writeback_tmp"`
	CommitLimit       uint64 `json:"commit_limit"`
	Committed_AS      uint64 `json:"committed_as"`
	VmallocTotal      uint64 `json:"vmalloc_total"`
	VmallocUsed       uint64 `json:"vmalloc_used"`
	VmallocChunk      uint64 `json:"vmalloc_chunk"`
	HardwareCorrupted uint64 `json:"hardware_corrupted"`
	AnonHugePages     uint64 `json:"anon_huge_pages"`
	HugePages_Total   uint64 `json:"huge_pages_total"`
	HugePages_Free    uint64 `json:"huge_pages_free"`
	HugePages_Rsvd    uint64 `json:"huge_pages_rsvd"`
	HugePages_Surp    uint64 `json:"huge_pages_surp"`
	Hugepagesize      uint64 `json:"hugepagesize"`
	DirectMap4k       uint64 `json:"direct_map_4k"`
	DirectMap2M       uint64 `json:"direct_map_2M"`
	DirectMap1G       uint64 `json:"direct_map_1G"`
}

type HugePagesInfo struct {
	NumPages uint64 `json:"numPages,omitempty"`
	PageSize uint64 `json:"pageSize,omitempty"`
}

func (i *HugePagesInfo) MemTotalBytes() uint64 {
	return i.PageSize * i.NumPages * 1024
}

// MemTotalBytes returns the mem info's total bytes.
func (i *MemInfo) MemTotalBytes() uint64 {
	return i.MemTotal * 1024
}

// MemUsageBytes returns the mem info's usage bytes.
func (i *MemInfo) MemUsageBytes() uint64 {
	// total - available
	return (i.MemTotal - i.MemAvailable) * 1024
}

// MemWithPageCacheUsageBytes returns the usage of mem with page cache bytes.
func (i *MemInfo) MemUsageWithPageCache() uint64 {
	// total - free
	return (i.MemTotal - i.MemFree) * 1024
}

// readMemInfo reads and parses the meminfo from the given file.
// If isNUMA=false, it parses each line without a prefix like "Node 0". Otherwise, it parses each line with the NUMA
// node prefix like "Node 0".
func readMemInfo(path string, isNUMA bool) (*MemInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := MemInfo{}
	// Maps a meminfo metric to its value (i.e. MemTotal --> 100000)
	statMap := make(map[string]uint64)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		var fields []string

		// trim the NUMA prefix:
		// e.g. `Node 1 MemTotal:       1048576 kB` -> `MemTotal:       1048576 kB`
		if isNUMA {
			fields = strings.SplitN(line, " ", 3)
			if len(fields) < 3 {
				continue
			}
			line = fields[2]
		}

		fields = strings.SplitN(line, ":", 2)
		if len(fields) < 2 {
			continue
		}
		valFields := strings.Fields(fields[1])
		val, _ := strconv.ParseUint(valFields[0], 10, 64)
		statMap[fields[0]] = val
	}

	elem := reflect.ValueOf(&info).Elem()
	typeOfElem := elem.Type()

	for i := 0; i < elem.NumField(); i++ {
		val, ok := statMap[typeOfElem.Field(i).Name]
		if ok {
			elem.Field(i).SetUint(val)
			continue
		}
		val, ok = statMap[typeOfElem.Field(i).Tag.Get("field")]
		if ok {
			elem.Field(i).SetUint(val)
		}
	}

	return &info, nil
}

func GetMemInfo() (*MemInfo, error) {
	memInfoPath := system.GetProcFilePath(system.ProcMemInfoName)
	memInfo, err := readMemInfo(memInfoPath, false)
	if err != nil {
		return nil, err
	}
	return memInfo, nil
}
