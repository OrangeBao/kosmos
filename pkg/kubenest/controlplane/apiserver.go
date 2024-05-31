package controlplane

import (
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/kosmos.io/kosmos/cmd/kubenest/operator/app/options"
	"github.com/kosmos.io/kosmos/pkg/kubenest/constants"
	"github.com/kosmos.io/kosmos/pkg/kubenest/manifest/controlplane/apiserver"
	"github.com/kosmos.io/kosmos/pkg/kubenest/util"
)

func EnsureVirtualClusterAPIServer(client clientset.Interface, name, namespace string, portMap map[string]int32, opt *options.KubeNestOptions) error {
	if err := installAPIServer(client, name, namespace, portMap, opt); err != nil {
		return fmt.Errorf("failed to install virtual cluster apiserver, err: %w", err)
	}
	return nil
}

func DeleteVirtualClusterAPIServer(client clientset.Interface, name, namespace string) error {
	deployName := fmt.Sprintf("%s-%s", name, "apiserver")
	if err := util.DeleteDeployment(client, deployName, namespace); err != nil {
		return errors.Wrapf(err, "Failed to delete deployment %s/%s", deployName, namespace)
	}
	return nil
}

func installAPIServer(client clientset.Interface, name, namespace string, portMap map[string]int32, opt *options.KubeNestOptions) error {
	imageRepository, imageVersion := util.GetImageMessage()
	clusterIp, err := util.GetEtcdServiceClusterIp(namespace, name+constants.EtcdSuffix, client)
	if err != nil {
		return nil
	}

	apiserverDeploymentBytes, err := util.ParseTemplate(apiserver.ApiserverDeployment, struct {
		DeploymentName, Namespace, ImageRepository, EtcdClientService, Version string
		ServiceSubnet, VirtualClusterCertsSecret, EtcdCertsSecret              string
		Replicas                                                               int
		EtcdListenClientPort                                                   int32
		ClusterPort                                                            int32
		AdmissionPlugins                                                       bool
	}{
		DeploymentName:            fmt.Sprintf("%s-%s", name, "apiserver"),
		Namespace:                 namespace,
		ImageRepository:           imageRepository,
		Version:                   imageVersion,
		EtcdClientService:         clusterIp,
		ServiceSubnet:             constants.ApiServerServiceSubnet,
		VirtualClusterCertsSecret: fmt.Sprintf("%s-%s", name, "cert"),
		EtcdCertsSecret:           fmt.Sprintf("%s-%s", name, "etcd-cert"),
		Replicas:                  opt.ApiServerReplicas,
		EtcdListenClientPort:      constants.ApiServerEtcdListenClientPort,
		ClusterPort:               portMap[constants.ApiServerPortKey],
		AdmissionPlugins:          opt.AdmissionPlugins,
	})
	if err != nil {
		return fmt.Errorf("error when parsing virtual cluster apiserver deployment template: %w", err)
	}

	apiserverDeployment := &appsv1.Deployment{}
	if err := yaml.Unmarshal([]byte(apiserverDeploymentBytes), apiserverDeployment); err != nil {
		return fmt.Errorf("error when decoding virtual cluster apiserver deployment: %w", err)
	}

	if err := util.CreateOrUpdateDeployment(client, apiserverDeployment); err != nil {
		return fmt.Errorf("error when creating deployment for %s, err: %w", apiserverDeployment.Name, err)
	}
	return nil
}
