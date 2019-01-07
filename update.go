// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/intelrdt"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

func i64Ptr(i int64) *int64   { return &i }
func u64Ptr(i uint64) *uint64 { return &i }
func u16Ptr(i uint16) *uint16 { return &i }

var updateCommand = cli.Command{
	Name:      "update",
	Usage:     "update container resource constraints",
	ArgsUsage: `<container-id>`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "resources, r",
			Value: "",
			Usage: `path to the file containing the resources to update or '-' to read from the standard input

The accepted format is as follow (unchanged values can be omitted):

{
  "memory": {
    "limit": 0,
    "reservation": 0,
    "swap": 0,
    "kernel": 0,
    "kernelTCP": 0
  },
  "cpu": {
    "shares": 0,
    "quota": 0,
    "period": 0,
    "realtimeRuntime": 0,
    "realtimePeriod": 0,
    "cpus": "",
    "mems": ""
  },
  "blockIO": {
    "weight": 0,
    "throttleReadBpsDevice": [{
      "major",
      "minor",
      "rate"
    }],
    "throttleWriteBpsDevice": [{
      "major",
      "minor",
      "rate"
    }],
    "throttleReadIOPSDevice": [{
      "major",
      "minor",
      "rate"
    }],
    "throttleWriteIOPSDevice": [{
      "major",
      "minor",
      "rate"
    }]
  }
}

Note: if data is to be read from a file or the standard input, all
other options are ignored.
`,
		},

		cli.IntFlag{
			Name:  "blkio-weight",
			Usage: "Specifies per cgroup weight, range is from 10 to 1000",
		},
		cli.StringSliceFlag{
			Name:  "device-read-bps",
			Usage: "Limit read rate (bytes per second) from a device, such as \"8:0 1048576\"",
		},
		cli.StringSliceFlag{
			Name:  "device-write-bps",
			Usage: "Limit write rate (bytes per second) to a device, such as \"8:0 1048576\"",
		},
		cli.StringSliceFlag{
			Name:  "device-read-iops",
			Usage: "Limit read rate (IO per second) from a device, such as \"8:0 5000\"",
		},
		cli.StringSliceFlag{
			Name:  "device-write-iops",
			Usage: "Limit write rate (IO per second) to a device, such as \"8:0 5000\"",
		},
		cli.StringFlag{
			Name:  "cpu-period",
			Usage: "CPU CFS period to be used for hardcapping (in usecs). 0 to use system default",
		},
		cli.StringFlag{
			Name:  "cpu-quota",
			Usage: "CPU CFS hardcap limit (in usecs). Allowed cpu time in a given period",
		},
		cli.StringFlag{
			Name:  "cpu-share",
			Usage: "CPU shares (relative weight vs. other containers)",
		},
		cli.StringFlag{
			Name:  "cpu-rt-period",
			Usage: "CPU realtime period to be used for hardcapping (in usecs). 0 to use system default",
		},
		cli.StringFlag{
			Name:  "cpu-rt-runtime",
			Usage: "CPU realtime hardcap limit (in usecs). Allowed cpu time in a given period",
		},
		cli.StringFlag{
			Name:  "cpuset-cpus",
			Usage: "CPU(s) to use",
		},
		cli.StringFlag{
			Name:  "cpuset-mems",
			Usage: "Memory node(s) to use",
		},
		cli.StringFlag{
			Name:  "kernel-memory",
			Usage: "Kernel memory limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "kernel-memory-tcp",
			Usage: "Kernel memory limit (in bytes) for tcp buffer",
		},
		cli.StringFlag{
			Name:  "memory",
			Usage: "Memory limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "memory-reservation",
			Usage: "Memory reservation or soft_limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "memory-swap",
			Usage: "Total memory usage (memory + swap); set '-1' to enable unlimited swap",
		},
		cli.IntFlag{
			Name:  "pids-limit",
			Usage: "Maximum number of pids allowed in the container",
		},
		cli.StringFlag{
			Name:  "l3-cache-schema",
			Usage: "The string of Intel RDT/CAT L3 cache schema",
		},
		cli.StringFlag{
			Name:  "mem-bw-schema",
			Usage: "The string of Intel RDT/MBA memory bandwidth schema",
		},
	},
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		container, err := getContainer(context)
		if err != nil {
			return err
		}

		r := specs.LinuxResources{
			Memory: &specs.LinuxMemory{
				Limit:       i64Ptr(0),
				Reservation: i64Ptr(0),
				Swap:        i64Ptr(0),
				Kernel:      i64Ptr(0),
				KernelTCP:   i64Ptr(0),
			},
			CPU: &specs.LinuxCPU{
				Shares:          u64Ptr(0),
				Quota:           i64Ptr(0),
				Period:          u64Ptr(0),
				RealtimeRuntime: i64Ptr(0),
				RealtimePeriod:  u64Ptr(0),
				Cpus:            "",
				Mems:            "",
			},
			BlockIO: &specs.LinuxBlockIO{
				Weight: u16Ptr(0),
			},
			Pids: &specs.LinuxPids{
				Limit: 0,
			},
		}

		config := container.Config()

		if in := context.String("resources"); in != "" {
			var (
				f   *os.File
				err error
			)
			switch in {
			case "-":
				f = os.Stdin
			default:
				f, err = os.Open(in)
				if err != nil {
					return err
				}
			}
			err = json.NewDecoder(f).Decode(&r)
			if err != nil {
				return err
			}
		} else {
			if val := context.Int("blkio-weight"); val != 0 {
				r.BlockIO.Weight = u16Ptr(uint16(val))
			}
			for _, pair := range []struct {
				opt  string
				dest *[]specs.LinuxThrottleDevice
			}{
				{"device-read-bps", &r.BlockIO.ThrottleReadBpsDevice},
				{"device-write-bps", &r.BlockIO.ThrottleWriteBpsDevice},
				{"device-read-iops", &r.BlockIO.ThrottleReadIOPSDevice},
				{"device-write-iops", &r.BlockIO.ThrottleWriteIOPSDevice},
			} {
				if val := context.StringSlice(pair.opt); len(val) > 0 {
					var err error
					*pair.dest, err = stringSliceToThrottleDevice(val)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
					}
				}
			}
			if val := context.String("cpuset-cpus"); val != "" {
				r.CPU.Cpus = val
			}
			if val := context.String("cpuset-mems"); val != "" {
				r.CPU.Mems = val
			}

			for _, pair := range []struct {
				opt  string
				dest *uint64
			}{

				{"cpu-period", r.CPU.Period},
				{"cpu-rt-period", r.CPU.RealtimePeriod},
				{"cpu-share", r.CPU.Shares},
			} {
				if val := context.String(pair.opt); val != "" {
					var err error
					*pair.dest, err = strconv.ParseUint(val, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
					}
				}
			}
			for _, pair := range []struct {
				opt  string
				dest *int64
			}{

				{"cpu-quota", r.CPU.Quota},
				{"cpu-rt-runtime", r.CPU.RealtimeRuntime},
			} {
				if val := context.String(pair.opt); val != "" {
					var err error
					*pair.dest, err = strconv.ParseInt(val, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
					}
				}
			}
			for _, pair := range []struct {
				opt  string
				dest *int64
			}{
				{"memory", r.Memory.Limit},
				{"memory-swap", r.Memory.Swap},
				{"kernel-memory", r.Memory.Kernel},
				{"kernel-memory-tcp", r.Memory.KernelTCP},
				{"memory-reservation", r.Memory.Reservation},
			} {
				if val := context.String(pair.opt); val != "" {
					var v int64

					if val != "-1" {
						v, err = units.RAMInBytes(val)
						if err != nil {
							return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
						}
					} else {
						v = -1
					}
					*pair.dest = v
				}
			}
			r.Pids.Limit = int64(context.Int("pids-limit"))
		}

		// Update the value
		config.Cgroups.Resources.BlkioWeight = *r.BlockIO.Weight
		config.Cgroups.Resources.CpuPeriod = *r.CPU.Period
		config.Cgroups.Resources.CpuQuota = *r.CPU.Quota
		config.Cgroups.Resources.CpuShares = *r.CPU.Shares
		config.Cgroups.Resources.CpuRtPeriod = *r.CPU.RealtimePeriod
		config.Cgroups.Resources.CpuRtRuntime = *r.CPU.RealtimeRuntime
		config.Cgroups.Resources.CpusetCpus = r.CPU.Cpus
		config.Cgroups.Resources.CpusetMems = r.CPU.Mems
		config.Cgroups.Resources.KernelMemory = *r.Memory.Kernel
		config.Cgroups.Resources.KernelMemoryTCP = *r.Memory.KernelTCP
		config.Cgroups.Resources.Memory = *r.Memory.Limit
		config.Cgroups.Resources.MemoryReservation = *r.Memory.Reservation
		config.Cgroups.Resources.MemorySwap = *r.Memory.Swap
		config.Cgroups.Resources.PidsLimit = r.Pids.Limit

		for _, pair := range []struct {
			dest *[]*configs.ThrottleDevice
			src  []specs.LinuxThrottleDevice
		}{
			{&config.Cgroups.Resources.BlkioThrottleReadBpsDevice, r.BlockIO.ThrottleReadBpsDevice},
			{&config.Cgroups.Resources.BlkioThrottleWriteBpsDevice, r.BlockIO.ThrottleWriteBpsDevice},
			{&config.Cgroups.Resources.BlkioThrottleReadIOPSDevice, r.BlockIO.ThrottleReadIOPSDevice},
			{&config.Cgroups.Resources.BlkioThrottleWriteIOPSDevice, r.BlockIO.ThrottleWriteIOPSDevice},
		} {
			for _, td := range pair.src {
				*pair.dest = append(*pair.dest, configs.NewThrottleDevice(td.Major, td.Minor, td.Rate))
			}
		}

		// Update Intel RDT
		l3CacheSchema := context.String("l3-cache-schema")
		memBwSchema := context.String("mem-bw-schema")
		if l3CacheSchema != "" && !intelrdt.IsCatEnabled() {
			return fmt.Errorf("Intel RDT/CAT: l3 cache schema is not enabled")
		}

		if memBwSchema != "" && !intelrdt.IsMbaEnabled() {
			return fmt.Errorf("Intel RDT/MBA: memory bandwidth schema is not enabled")
		}

		if l3CacheSchema != "" || memBwSchema != "" {
			// If intelRdt is not specified in original configuration, we just don't
			// Apply() to create intelRdt group or attach tasks for this container.
			// In update command, we could re-enable through IntelRdtManager.Apply()
			// and then update intelrdt constraint.
			if config.IntelRdt == nil {
				state, err := container.State()
				if err != nil {
					return err
				}
				config.IntelRdt = &configs.IntelRdt{}
				intelRdtManager := intelrdt.IntelRdtManager{
					Config: &config,
					Id:     container.ID(),
					Path:   state.IntelRdtPath,
				}
				if err := intelRdtManager.Apply(state.InitProcessPid); err != nil {
					return err
				}
			}
			config.IntelRdt.L3CacheSchema = l3CacheSchema
			config.IntelRdt.MemBwSchema = memBwSchema
		}

		return container.Set(config)
	},
}

func stringSliceToThrottleDevice(ss []string) ([]specs.LinuxThrottleDevice, error) {
	tds := make([]specs.LinuxThrottleDevice, 0, len(ss))
	for _, v := range ss {
		parts := strings.SplitN(strings.TrimSpace(v), " ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("failed to parse throttle device %s: the expected format is 'major:minor rate'", v)
		}
		devs := strings.SplitN(parts[0], ":", 2)
		if len(devs) != 2 {
			return nil, fmt.Errorf("failed to parse throttle device %s: the expected format is 'major:minor rate'", v)
		}

		td := specs.LinuxThrottleDevice{}
		for _, item := range []struct {
			value string
			dest  *int64
		}{
			{devs[0], &td.Major},
			{devs[1], &td.Minor},
		} {
			v, err := strconv.ParseInt(item.value, 10, 64)
			if err != nil {
				return nil, err
			}
			*item.dest = v
		}
		rate, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		td.Rate = rate

		tds = append(tds, td)
	}
	return tds, nil
}
