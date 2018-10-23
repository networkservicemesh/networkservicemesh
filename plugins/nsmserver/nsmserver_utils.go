package nsmserver

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getContainerID(k8s *kubernetes.Clientset, pn, ns string) (string, error) {
	pl, err := k8s.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, p := range pl.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, pn) {
			// Two cases main container is in Running state and in Pending
			// Pending can inidcate that Init container is still running
			if p.Status.Phase == v1.PodRunning {
				return strings.Split(p.Status.ContainerStatuses[0].ContainerID, "://")[1][:12], nil
			}
			if p.Status.Phase == v1.PodPending {
				// Check if we have Init containers
				if p.Status.InitContainerStatuses != nil {
					for _, i := range p.Status.InitContainerStatuses {
						if i.State.Running != nil {
							return strings.Split(i.ContainerID, "://")[1][:12], nil
						}
					}
				}
			}
			return "", fmt.Errorf("none of containers of pod %s/%s is in running state", p.ObjectMeta.Namespace, p.ObjectMeta.Name)
		}
	}

	return "", fmt.Errorf("pod %s/%s not found", ns, pn)
}

func getPidForContainer(id string) (int, error) {
	pid := 0
	// memory is chosen randomly, any cgroup used by docker works
	cgroupType := "memory"

	cgroupRoot, err := findCgroupMountpoint(cgroupType)
	if err != nil {
		return pid, err
	}
	cgroupThis, err := getThisCgroup(cgroupType)
	if err != nil {
		return pid, err
	}

	id += "*"

	attempts := []string{
		filepath.Join(cgroupRoot, cgroupThis, id, "tasks"),
		// With more recent lxc versions use, cgroup will be in lxc/
		filepath.Join(cgroupRoot, cgroupThis, "lxc", id, "tasks"),
		// With more recent docker, cgroup will be in docker/
		filepath.Join(cgroupRoot, cgroupThis, "docker", id, "tasks"),
		// Even more recent docker versions under systemd use docker-<id>.scope/
		filepath.Join(cgroupRoot, "system.slice", "docker-"+id+".scope", "tasks"),
		// Even more recent docker versions under cgroup/systemd/docker/<id>/
		filepath.Join(cgroupRoot, "..", "systemd", "docker", id, "tasks"),
		// Kubernetes with docker and CNI is even more different
		filepath.Join(cgroupRoot, "..", "systemd", "kubepods", "*", "pod*", id, "tasks"),
		// Another flavor of containers location in recent kubernetes 1.11+
		filepath.Join(cgroupRoot, cgroupThis, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
		// When runs inside of a container with recent kubernetes 1.11+
		filepath.Join(cgroupRoot, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
		// For Docker in Docker container goes right under cgroupRoot
		filepath.Join(cgroupRoot, id+"*", "tasks"),
	}

	var filename string
	for _, attempt := range attempts {
		filenames, err := filepath.Glob(attempt)
		if err != nil {
			return pid, err
		}
		if filenames == nil {
			continue
		}
		if len(filenames) > 1 {
			return pid, fmt.Errorf("Ambiguous id supplied: %v", filenames)
		}
		if len(filenames) == 1 {
			filename = filenames[0]
			break
		}
	}

	if filename == "" {
		return pid, fmt.Errorf("Unable to find container: %v", id[:len(id)-1])
	}

	output, err := ioutil.ReadFile(filename)
	if err != nil {
		return pid, err
	}

	result := strings.Split(string(output), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return pid, fmt.Errorf("No pid found for container")
	}

	pid, err = strconv.Atoi(result[0])
	if err != nil {
		return pid, fmt.Errorf("Invalid pid '%s': %s", result[0], err)
	}

	return pid, nil
}

// borrowed from docker/utils/utils.go
func findCgroupMountpoint(cgroupType string) (string, error) {
	output, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}

	// /proc/mounts has 6 fields per line, one mount per line, e.g.
	// cgroup /sys/fs/cgroup/devices cgroup rw,relatime,devices 0 0
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, " ")
		if len(parts) == 6 && parts[2] == "cgroup" {
			for _, opt := range strings.Split(parts[3], ",") {
				if opt == cgroupType {
					return parts[1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("cgroup mountpoint not found for %s", cgroupType)
}

// Returns the relative path to the cgroup docker is running in.
// borrowed from docker/utils/utils.go
// modified to get the docker pid instead of using /proc/self
func getThisCgroup(cgroupType string) (string, error) {
	dockerpid, err := ioutil.ReadFile("/var/run/docker.pid")
	if err != nil {
		return "", err
	}
	result := strings.Split(string(dockerpid), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return "", fmt.Errorf("docker pid not found in /var/run/docker.pid")
	}
	pid, err := strconv.Atoi(result[0])
	if err != nil {
		return "", err
	}
	output, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, ":")
		// any type used by docker should work
		if parts[1] == cgroupType {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("cgroup '%s' not found in /proc/%d/cgroup", cgroupType, pid)
}
