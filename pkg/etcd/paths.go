package etcd

import (
	"fmt"
	"strings"
)

// GenerateTaskPrefixPath 生成特定集群和任务类型的 Etcd 路径前缀
// 格式: <basePath>/<cluster_name>/pending/<task_name>/
func GenerateTaskPrefixPath(basePath, clusterName, taskName string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	if basePath == "" {
		basePath = "/trend"
	}
	return fmt.Sprintf("%s/%s/pending/%s/", basePath, clusterName, taskName)
}

// GenerateTaskPath 生成具体任务的完整 Etcd 路径
func GenerateTaskPath(basePath, clusterName, taskName, taskID string) string {
	return GenerateTaskPrefixPath(basePath, clusterName, taskName) + taskID
}
