package deviceplugin

const (
	nsmServerSocket = "nsm.server.io.sock"
	nsmClientSocket = "nsm.client.io.sock"
)

func containerDeviceDirectory(deviceId string) string {
	return BaseDir
}

func hostDeviceDirectory(deviceId string) string {
	return BaseDir + deviceId + "/"
}

func localDeviceDirectory(deviceId string) string {
	return hostDeviceDirectory(deviceId)
}

func containerServerSocketFile(deviceId string) string {
	return containerDeviceDirectory(deviceId) + nsmServerSocket
}

func hostServerSocketFile(deviceId string) string {
	return hostDeviceDirectory(deviceId) + nsmServerSocket
}

func localServerSocketFile(deviceId string) string {
	return hostServerSocketFile(deviceId)
}

func containerClientSocketFile(deviceId string) string {
	return containerDeviceDirectory(deviceId) + nsmClientSocket
}

func hostClientSocketFile(deviceId string) string {
	return hostDeviceDirectory(deviceId) + nsmClientSocket
}

func localClientSocketFile(deviceId string) string {
	return hostClientSocketFile(deviceId)
}
