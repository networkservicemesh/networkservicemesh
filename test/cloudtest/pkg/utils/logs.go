package utils

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"strings"
)

//TODO: cleanup and benchmark this
func CollectLogs(content string) []ContainerLogs {
	result := make([]ContainerLogs, 0, 4)
	index := strings.Index(content, kubetest.StartLogsOf)

	for index != -1 {
		name := parseName(content, index+len(kubetest.StartLogsOf))
		endIndex := strings.Index(content, kubetest.EndLogsOf)
		if endIndex == -1 {
			return nil
		}
		l1 := len(name) + len(kubetest.StartLogsOf)
		l2 := len(name) + len(kubetest.EndLogsOf)
		sideSize1 := (kubetest.MaxTransactionLineWidth - l1) / 2
		sideSize2 := (kubetest.MaxTransactionLineWidth - l2) / 2
		startIndex := index + kubetest.MaxTransactionLineWidth + sideSize1 + 2 + l1
		endIndex -= kubetest.MaxTransactionLineWidth + sideSize2 + 2
		result = append(result, ContainerLogs{
			ContainerName: name,
			Logs:          content[startIndex : endIndex+1],
		})
		content = content[endIndex+3*kubetest.MaxTransactionLineWidth+3:]
		index = strings.Index(content, kubetest.StartLogsOf)
	}

	return result
}

func parseName(str string, index int) string {
	for l := index; l < len(str); l++ {
		if str[l] == kubetest.TransactionLogUnit {
			return str[index:l]
		}
	}
	return str[index:]
}

type ContainerLogs struct {
	ContainerName string
	Logs          string
}
