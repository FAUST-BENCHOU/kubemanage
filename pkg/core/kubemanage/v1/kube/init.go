package kube

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/noovertime7/kubemanage/cmd/app/config"
	"github.com/noovertime7/kubemanage/pkg/logger"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var K8s k8s

type k8s struct {
	Config    *rest.Config
	ClientSet *kubernetes.Clientset
}

func (k *k8s) Init() error {
	var err error
	var restConfig *rest.Config
	var kubeConfigPath string

	// 优先级：配置文件 > 命令行参数 > 默认路径
	if config.SysConfig != nil && config.SysConfig.Default.KubernetesConfigFile != "" {
		// 从配置文件读取
		kubeConfigPath = config.SysConfig.Default.KubernetesConfigFile
	} else {
		// 使用命令行参数或默认路径
		var kubeConfig *string
		if home := homeDir(); home != "" {
			kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		kubeConfigPath = *kubeConfig
	}

	// 使用 ServiceAccount 创建集群配置（InCluster模式）
	if restConfig, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置
		if restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath); err != nil {
			return err
		}
	}

	// 创建 clientSet
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	log := logger.New(logger.LG)
	log.Info("获取k8s clientSet 成功")
	k.ClientSet = clientSet
	k.Config = restConfig
	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
