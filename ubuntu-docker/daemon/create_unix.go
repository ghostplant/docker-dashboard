// +build !windows

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/container"
	"github.com/docker/docker/pkg/stringid"
	containertypes "github.com/docker/engine-api/types/container"
	"github.com/docker/go-units"
	"github.com/opencontainers/runc/libcontainer/label"
)

func (daemon *Daemon) normalizeStandaloneAndClusterSettings(container *container.Container, config *containertypes.Config, hostConfig *containertypes.HostConfig) error {

	if v, ok := config.Labels["com.docker.resource.climit.pids"]; ok {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot recognize pids value: %s", v)
		}
		hostConfig.Resources.PidsLimit = val
	}

	if v, ok := config.Labels["com.docker.resource.climit.cpu_shares"]; ok {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot recognize cpu_shares value: %s", v)
		}
		hostConfig.Resources.CPUShares = val
	}

	if v, ok := config.Labels["com.docker.resource.climit.cpu_pquota"]; ok {
		pquota := strings.SplitN(v, "/", 2)
		if len(pquota) != 2 {
			return fmt.Errorf("cannot recognize cpu_pquota value: %s", v)
		}
		quota, err1 := strconv.ParseInt(pquota[0], 10, 64)
		period, err2 := strconv.ParseInt(pquota[1], 10, 64)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("cannot recognize cpu_pquota value: %s", v)
		}
		hostConfig.Resources.CPUQuota = quota
		hostConfig.Resources.CPUPeriod = period
	}

	if v, ok := config.Labels["com.docker.resource.climit.blkio_weight"]; ok {
		val, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return fmt.Errorf("cannot recognize blkio_weight value: %s", v)
		}
		hostConfig.Resources.BlkioWeight = uint16(val)
	}

	if v, ok := config.Labels["com.docker.resource.climit.memswap"]; ok {
		val, err := units.RAMInBytes(v)
		if err != nil {
			return fmt.Errorf("cannot recognize memswap value: %s", v)
		}
		hostConfig.Resources.MemorySwap = val
	}

	if v, ok := config.Labels["com.docker.resource.climit.memkrnl"]; ok {
		val, err := units.RAMInBytes(v)
		if err != nil {
			return fmt.Errorf("cannot recognize memkrnl value: %s", v)
		}
		hostConfig.Resources.KernelMemory = val
	}

	if v, ok := config.Labels["com.docker.resource.climit.memsoft"]; ok {
		val, err := units.RAMInBytes(v)
		if err != nil {
			return fmt.Errorf("cannot recognize memsoft value: %s", v)
		}
		hostConfig.Resources.MemoryReservation = val
	}

	if v, ok := config.Labels["com.docker.resource.climit.memhard"]; ok {
		val, err := units.RAMInBytes(v)
		if err != nil {
			return fmt.Errorf("cannot recognize memhard value: %s", v)
		}
		hostConfig.Resources.Memory = val
	}


	if v, ok := config.Labels["com.docker.resource.climit.oom_kill_disable"]; ok {
		val, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("cannot recognize oom_kill_disable value: %s", v)
		}
		hostConfig.Resources.OomKillDisable = &val
	}

	if v, ok := config.Labels["com.docker.resource.ulimit.opts"]; ok {
		// e.g. "nofile=40960"
		opts := strings.Split(v, ",")
		if hostConfig.Resources.Ulimits == nil {
			hostConfig.Resources.Ulimits = []*units.Ulimit{}
		}
		for id := range opts {
			opt, err := units.ParseUlimit(opts[id])
			if err != nil {
				return err
			}
			hostConfig.Resources.Ulimits = append(hostConfig.Resources.Ulimits, opt)
		}
	}

	/*if v, ok := config.Labels["com.docker.resource.ulimit.vmkb"]; ok {
		val, err := units.RAMInBytes(v)
		if err != nil {
			return fmt.Errorf("cannot recognize ulimit.vmkb value: %s", v)
		}
		val >>= 10
		asLimit := &units.Ulimit{Name: "as", Soft: val, Hard: val}
		hostConfig.Resources.Ulimits = append(hostConfig.Resources.Ulimits, asLimit)
	}*/

	if v, ok := config.Labels["com.docker.storage.opts"]; ok {
		opts := strings.Split(v, ",")
		if hostConfig.StorageOpt == nil {
			hostConfig.StorageOpt = make(map[string]string)
		}
		for id := range opts {
			pair := strings.SplitN(opts[id], "=", 2)
			if len(pair) != 2 {
				return fmt.Errorf("cannot recognize storage option value: %s", pair)
			}
			hostConfig.StorageOpt[pair[0]] = pair[1]
		}
	}

	return nil
}

// createContainerPlatformSpecificSettings performs platform specific container create functionality
func (daemon *Daemon) createContainerPlatformSpecificSettings(container *container.Container, config *containertypes.Config, hostConfig *containertypes.HostConfig) error {
	if err := daemon.Mount(container); err != nil {
		return err
	}
	defer daemon.Unmount(container)

	rootUID, rootGID := daemon.GetRemappedUIDGID()
	if err := container.SetupWorkingDirectory(rootUID, rootGID); err != nil {
		return err
	}

	var defaultDriver = hostConfig.VolumeDriver
	defaultConfig := false
	if driver, ok := config.Labels["com.github.private.volume.default.driver"]; ok {
		defaultDriver = driver
		defaultConfig = true
	}
	var defaultOpts map[string]string
	if opts, ok := config.Labels["com.github.private.volume.default.opts"]; ok {
		defaultOpts = make(map[string]string)
		rawOpts := strings.Split(opts, ",")
		for id := range rawOpts {
			pair := strings.SplitN(rawOpts[id], "=", 2)
			if len(pair) != 2 {
				return fmt.Errorf("Unrecognised default opts: %s", rawOpts[id])
			}
			defaultOpts[pair[0]] = pair[1]
		}
		defaultConfig = true
	}

	if disabled, ok := config.Labels["com.github.private.volume.default.disabled"]; ok && disabled == "true" {
		config.Volumes = make(map[string]struct{})
	} else if ext, ok := config.Labels["com.github.private.volume.extension.paths"]; ok {
		paths := strings.Split(ext, ",")
		for id := range paths {
			if len(paths[id]) > 0 && paths[id][0] == '/' {
				config.Volumes[paths[id]] = struct{}{}
			}
		}
	}

	for spec := range config.Volumes {
		name := stringid.GenerateNonCryptoID()
		if defaultConfig {
			if defaultName, ok := config.Labels["com.github.private.volume.default.name"]; ok {
				name = defaultName
			} else if serviceName, ok := config.Labels["com.docker.swarm.service.name"]; ok {
				name = fmt.Sprintf("%s..%x", serviceName, spec)
			}
		}
		destination := filepath.Clean(spec)

		// Skip volumes for which we already have something mounted on that
		// destination because of a --volume-from.
		if container.IsDestinationMounted(destination) {
			continue
		}
		path, err := container.GetResourcePath(destination)
		if err != nil {
			return err
		}

		stat, err := os.Stat(path)
		if err == nil && !stat.IsDir() {
			return fmt.Errorf("cannot mount volume over existing file, file exists %s", path)
		}

		v, err := daemon.volumes.CreateWithRef(name, defaultDriver, container.ID, defaultOpts, nil)
		if err != nil {
			return err
		}

		if err := label.Relabel(v.Path(), container.MountLabel, true); err != nil {
			return err
		}

		container.AddMountPointWithVolume(destination, v, true)
	}
	return daemon.populateVolumes(container)
}

// populateVolumes copies data from the container's rootfs into the volume for non-binds.
// this is only called when the container is created.
func (daemon *Daemon) populateVolumes(c *container.Container) error {
	for _, mnt := range c.MountPoints {
		if !mnt.CopyData || mnt.Volume == nil {
			continue
		}

		logrus.Debugf("copying image data from %s:%s, to %s", c.ID, mnt.Destination, mnt.Name)
		if err := c.CopyImagePathContent(mnt.Volume, mnt.Destination); err != nil {
			return err
		}
	}
	return nil
}
