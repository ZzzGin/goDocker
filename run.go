package main

import (
	"encoding/json"
	"fmt"
	"godocker/cgroups"
	"godocker/cgroups/subsystems"
	"godocker/container"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, volume string, containerName string, imageName string, envSlice []string) {
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}
	
	parent, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, envSlice)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	//record container info
	containerName, err := recordContainerInfo(parent.Process.Pid, comArray, containerName, volume)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}

	// use mydocker-cgroup as cgroup name
	cgroupManager := cgroups.NewCgroupManager("godocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	sendInitCommand(comArray, writePipe)

	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName) // be carefule!!!
	}
	os.Exit(0)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

func recordContainerInfo(containerPID int, commandArray []string, containerName string, volume string) (string, error) {
	id := randStringBytes(10)
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, "")
	if containerName == "" {
		containerName = id
	}
	containerInfo := &container.ContainerInfo{
		Id:          id,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreatedTime: createTime,
		Status:      container.RUNNING,
		Name:        containerName,
		Volume:      volume,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirUrl, err)
		return "", err
	}
	fileName := dirUrl + "/" + container.ConfigName
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
}

func deleteContainerInfo(containerId string) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
