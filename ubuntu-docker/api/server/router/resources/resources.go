package resources

import (
	//"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	//"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/server/httputils"
	"golang.org/x/net/context"

	"github.com/docker/docker/api/server/router"
	"github.com/docker/docker/daemon"
	"github.com/docker/docker/daemon/cluster"
)

type resourcesRouter struct {
	backend         *daemon.Daemon
	clusterProvider *cluster.Cluster
	routes          []router.Route
}

type resourceItem struct {
	ContainerID     string

	MemoryUsage     int64
	MemorySwapUsage int64
	MemoryKrnlUsage int64
	MemoryPeak      int64
	MemorySwapPeak  int64
	MemoryKrnlPeak  int64
	MemoryLimit     int64
	MemorySwapLimit int64
	MemorySoftLimit int64
	MemoryKrnlLimit int64

	PidsLimit       int64
	PidsUsage       int64

	CpuUsage        int64
	CpuShares       int64
	CpuPeriod       int64
	CpuQuota        int64

	BlkioWeight     uint16

	OomStatus       string

/*	DiskQuota       int64
	DiskUsage       int64

	DefVolumeDriver string
	DefVolumeType   string
	DefVolumeOpts   string */
}

type resourcesInfo struct {
	DiskFreeSizeMB  int64
	DiskFreeRatio   float64

	CpuLogicNumber  int64
	CpuUptimeLoad   []float64

	MemoryTotel     int64
	MemoryAvail     int64
	MemoryBuffered  int64
	MemorySwapTotal int64
	MemorySwapAvail int64

	Resources      []*resourceItem
}

func NewRouter(b *daemon.Daemon, c *cluster.Cluster) router.Router {
	r := &resourcesRouter{
		backend:         b,
		clusterProvider: c,
	}

	r.routes = []router.Route{
		router.NewGetRoute("/resources", r.getResources),
	}

	return r
}

func (r *resourcesRouter) Routes() []router.Route {
	return r.routes
}

func ReadCgroupValue(id, subsys, key string) string {
	fp, err := os.Open("/sys/fs/cgroup/" + subsys + "/docker/" + id + "/" + key)
	if err != nil {
		return ""
	}
	defer fp.Close()
	data, err := ioutil.ReadAll(fp)
	if err != nil {
		return ""
	}
	size := len(data)
	if data[size - 1] == '\n' {
		return string(data[:size - 1])
	} else {
		return string(data)
	}
}

func ToInt64(val string, def int64) int64 {
	if v, err := strconv.ParseInt(val, 10, 64); err == nil {
		return v
	}
	return def
}

func (r *resourcesRouter) getResources(ctx context.Context, w http.ResponseWriter, req *http.Request, vars map[string]string) error {
	info, _ := r.backend.SystemInfo()
	if out, err := exec.Command("df", "-m", info.DockerRootDir).Output(); err == nil {
		parts := strings.Split(string(out[:]), "\n")
		if len(parts) > 1 {
			section := 0
			freeSize := int64(-1)
			freeRatio := float64(-1.0)
			for _, val := range strings.Split(parts[1], " ") {
				if len(val) == 0 {
					continue
				}
				section++
				if v, err := strconv.ParseInt(val, 10, 64); section == 4 && err == nil {
					freeSize = v
				} else if v, err := strconv.ParseFloat(strings.SplitN(val, "%", 2)[0], 64); section == 5 && err == nil {
					freeRatio = v * 0.01
				}
			}
			if freeSize >= 0 && freeRatio >= 0.0 {
				resources := []*resourceItem{}
				files, _ := ioutil.ReadDir("/sys/fs/cgroup/memory/docker")
				for _, fi := range files {
					if fi.IsDir() {
						id := fi.Name()
						resources = append(resources, &resourceItem{
							ContainerID:     id,
							MemoryUsage:     ToInt64(ReadCgroupValue(id, "memory", "memory.usage_in_bytes"), -1),
							MemorySwapUsage: ToInt64(ReadCgroupValue(id, "memory", "memory.memsw.usage_in_bytes"), -1),
							MemoryKrnlUsage: ToInt64(ReadCgroupValue(id, "memory", "memory.kmem.usage_in_bytes"), -1),
							MemoryPeak:      ToInt64(ReadCgroupValue(id, "memory", "memory.max_usage_in_bytes"), -1),
							MemorySwapPeak:  ToInt64(ReadCgroupValue(id, "memory", "memory.memsw.max_usage_in_bytes"), -1),
							MemoryKrnlPeak:  ToInt64(ReadCgroupValue(id, "memory", "memory.kmem.max_usage_in_bytes"), -1),
							MemoryLimit:     ToInt64(ReadCgroupValue(id, "memory", "memory.limit_in_bytes"), -1),
							MemorySwapLimit: ToInt64(ReadCgroupValue(id, "memory", "memory.memsw.limit_in_bytes"), -1),
							MemoryKrnlLimit: ToInt64(ReadCgroupValue(id, "memory", "memory.kmem.limit_in_bytes"), -1),
							MemorySoftLimit: ToInt64(ReadCgroupValue(id, "memory", "memory.soft_limit_in_bytes"), -1),
							PidsUsage:       ToInt64(ReadCgroupValue(id, "pids", "pids.current"), -1),
							PidsLimit:       ToInt64(ReadCgroupValue(id, "pids", "pids.max"), -1),
							CpuUsage:        ToInt64(ReadCgroupValue(id, "cpu", "cpuacct.usage"), -1),
							CpuShares:       ToInt64(ReadCgroupValue(id, "cpu", "cpu.shares"), -1),
							CpuPeriod:       ToInt64(ReadCgroupValue(id, "cpu", "cpu.cfs_period_us"), -1),
							CpuQuota:        ToInt64(ReadCgroupValue(id, "cpu", "cpu.cfs_quota_us"), -1),
							BlkioWeight:     uint16(ToInt64(ReadCgroupValue(id, "blkio", "blkio.weight"), -1)),
							OomStatus:       ReadCgroupValue(id, "memory", "memory.oom_control"),
						})
					}
				}

				result := &resourcesInfo{
					DiskFreeSizeMB: freeSize,
					DiskFreeRatio:  freeRatio,
					Resources:      resources,
				}
				return httputils.WriteJSON(w, http.StatusOK, result)
			}
		}
	}
	return httputils.WriteJSON(w, http.StatusInternalServerError, "")
}

