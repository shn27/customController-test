package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	kubeConfigFilePath := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "kubeconfig file path")
	flag.Parse()
	clientset := InitializeClientGo(*kubeConfigFilePath)
	ch := make(chan struct{})
	informers := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	c := newController(clientset, informers.Apps().V1().Deployments())
	informers.Start(ch)
	c.run(ch)
	fmt.Println(informers)
}

func InitializeClientGo(kubeConfigFilePath string) *kubernetes.Clientset {
	if pathExists := func(path string) bool {
		if _, err := os.Stat(path); err != nil {
			log.Println("Path " + path + " does not exist. Loading incluster config")
			return false
		}
		return true
	}(kubeConfigFilePath); !pathExists {
		kubeConfigFilePath = ""
		log.Println("Kubeconfig path doesn't exist. using the inClusterConfig")
	} else {
		log.Println("Loading kubeconfig file " + kubeConfigFilePath)
	}

	var err error
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFilePath)
	if err != nil {
		log.Fatal("error loading kubeconfig ", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("error creating clientset", err)
	}

	log.Println("Successfully loaded kubeconfig")
	return clientSet
}
