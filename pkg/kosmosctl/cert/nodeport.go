package cert

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kosmos.io/kosmos/pkg/kubenest/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var PortExample = templates.Examples(i18n.T(`
     # change port, e.g:
     kosmosctl renew nodeport --kubeconfig=xxxx  --namespace=xxxx --name=xxxx --agent-user=xxxx --agent-pass=xxxx
`))

type NodePortCmdOptions struct {
	CertOptions CertOptions
}

func NewCmdNodePortCert() *cobra.Command {
	o := &NodePortCmdOptions{}
	cmd := &cobra.Command{
		Use:                   "nodeport",
		Short:                 i18n.T("change hostnetwork to nodeport for virtual cluster."),
		Long:                  "",
		Example:               PortExample,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctlutil.CheckErr(o.Complete())
			ctlutil.CheckErr(o.Validate())
			ctlutil.CheckErr(o.Run())
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&o.CertOptions.Namespace, "namespace", "e", "", "namespace of vc")
	flags.StringVarP(&o.CertOptions.Name, "name", "n", "", "name of vc")
	flags.StringVarP(&o.CertOptions.KubeconfigPath, "kubeconfig", "k", "", "kubeconfig path of host cluster")
	flags.StringVarP(&o.CertOptions.WebUser, "agent-user", "u", "", "user of node agent")
	flags.StringVarP(&o.CertOptions.WebPass, "agent-pass", "p", "", "password of node agent")
	return cmd
}

func (o *NodePortCmdOptions) Complete() (err error) {
	return nil
}

func (o *NodePortCmdOptions) Validate() error {
	if len(o.CertOptions.WebPass) == 0 {
		return fmt.Errorf("web pass is required")
	}

	if len(o.CertOptions.WebUser) == 0 {
		return fmt.Errorf("use pass is required")
	}
	if len(o.CertOptions.KubeconfigPath) == 0 {
		return fmt.Errorf("kubeconfig path is required")
	}
	if len(o.CertOptions.Namespace) == 0 {
		return fmt.Errorf("namespace is required")
	}
	if len(o.CertOptions.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (o *NodePortCmdOptions) initEnv() {
	os.Setenv("KUBECONFIG", o.CertOptions.KubeconfigPath)
	os.Setenv("WEB_USER", o.CertOptions.WebUser)
	os.Setenv("WEB_PASS", o.CertOptions.WebPass)
}

func (o *NodePortCmdOptions) Run() error {
	r, err := NewCertOption(&o.CertOptions)
	o.initEnv()
	if err != nil {
		return err
	}

	err = RunTask([]TaskFunc{
		RunCheckEnvironmentForTls,
		// 获取 apiserver-port的Nodeort的端口  记为A ， 通过vc中的字段获取agentPort的端口， 通过修改svc [vc-name]-apiserver，来获取agentPort对应的nodePort端口。记为B
		// 更新 secrets   [vc-name]-admin-config-clusterip ，端口改为 A
		GetNodePortForApiServer,
		UpdateVCAdminConfigClusteripSecret,
		// 更新 专属集群的apiserver的deployment  等待服务就绪
		UpdateApiServerDeployment,
		// 重启 coredns， kube-controller-manager，scheduler  等待服务就绪
		RestartVirtualControlPlanePodForSLT,
		// 删除专属集群中的默认namespace下的名为kubernetes的endpoints
		// 修改 专属集群中 konnectivity-server的endpoints的端口和ip为 B 和master节点
		// 检查隧道服务是否就绪

	}, r)
	if err != nil {
		return err
	}
	klog.Infof("############ renew cert success!!!!")
	return nil
}

func RunCheckEnvironmentForTls(data *Option) error {
	// TODO check 是否已经是Nodeort了
	vc := data.VirtualCluster()
	klog.Infof("try to run command kubectl")
	namespace := vc.GetNamespace()
	name := vc.GetName()
	commands := [][]string{
		{
			"--kubeconfig",
			HostClusterConfigPath(),
			"-n", namespace,
			"get",
			"vc",
			name,
		},
	}

	for _, args := range commands {
		klog.InfoS("run command:", strings.Join(args, " "))
		if err := runKubectlCommand(args...); err != nil {
			klog.InfoS("run command failed:", err)
			return err
		}
	}

	return nil
}

func GetNodePortForApiServer(data *Option) error {
	vc := data.VirtualCluster()

	currentApiSvc, err := data.RemoteClient().CoreV1().Services(vc.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-apiserver", vc.GetName()), metav1.GetOptions{})
	if err != nil {
		return err
	}

	agentport := vc.Status.PortMap["apiserver-network-proxy-agent-port"]

	apisvc := currentApiSvc.DeepCopy()
	var apiserverPort int32
	for i, port := range apisvc.Spec.Ports {
		if port.Name == "client" {
			apiserverPort = port.NodePort
			apisvc.Spec.Ports[i].Port = apiserverPort
			apisvc.Spec.Ports[i].TargetPort = intstr.IntOrString{
				IntVal: apiserverPort,
			}
		}
		if port.Name == "agentport" {
			apisvc.Spec.Ports[i].NodePort = agentport
			apisvc.Spec.Ports[i].Port = agentport
			apisvc.Spec.Ports[i].TargetPort = intstr.IntOrString{
				IntVal: agentport,
			}
		}
	}

	if len(apisvc.Spec.Ports) == 1 {
		apisvc.Spec.Ports = append(apisvc.Spec.Ports, corev1.ServicePort{
			Name: "agentport",
			Port: agentport,
			TargetPort: intstr.IntOrString{
				IntVal: agentport,
			},
			NodePort: agentport,
		})
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apisvc.ResourceVersion = ""
		_, err = data.RemoteClient().CoreV1().Services(vc.GetNamespace()).Update(context.TODO(), apisvc, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		return err
	}

	data.tmpValueMap["client"] = apiserverPort
	data.tmpValueMap["agentport"] = agentport
	return nil
}

func UpdateApiServerDeployment(data *Option) error {
	vc := data.VirtualCluster()

	deploymentName := fmt.Sprintf("%s-apiserver", vc.GetName())
	currentApiDeployment, err := data.RemoteClient().AppsV1().Deployments(vc.GetNamespace()).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	val, ok := data.tmpValueMap["client"].(int)
	if !ok {
		return fmt.Errorf("get client port from map failed")
	}
	apiserverPort := int32(val)

	apiDeployment := currentApiDeployment.DeepCopy()

	for i, container := range apiDeployment.Spec.Template.Spec.Containers {
		if container.Name == "kube-apiserver" {
			apiDeployment.Spec.Template.Spec.Containers[i].LivenessProbe.HTTPGet.Port = intstr.IntOrString{
				IntVal: apiserverPort,
			}

			apiDeployment.Spec.Template.Spec.Containers[i].ReadinessProbe.HTTPGet.Port = intstr.IntOrString{
				IntVal: apiserverPort,
			}

			for j, port := range apiDeployment.Spec.Template.Spec.Containers[i].Ports {
				if port.Name == "http" {
					apiDeployment.Spec.Template.Spec.Containers[i].Ports[j].ContainerPort = apiserverPort
				}
			}

			for j, cmdstr := range container.Command {
				if strings.Contains(cmdstr, "--secure-port") {
					apiDeployment.Spec.Template.Spec.Containers[i].Command[j] = fmt.Sprintf("--secure-port=%d", apiserverPort)
				}
				if strings.Contains(cmdstr, "--advertise-address=") {
					apiDeployment.Spec.Template.Spec.Containers[i].Command[j] = "--advertise-address=$(HOSTIP)"
				}
			}
			for j, env := range container.Env {
				if env.Name == "PODIP" {
					apiDeployment.Spec.Template.Spec.Containers[i].Env[j].ValueFrom.FieldRef.FieldPath = "status.hostIP"
				}
			}
		}
	}

	return WaitDaemonsetReady(data.remoteClient, vc.GetNamespace(), deploymentName)
}

func RestartVirtualControlPlanePodForSLT(data *Option) error {
	klog.Infof("restart control-plane pod in host cluster")
	vc := data.VirtualCluster()

	namespace := vc.GetNamespace()
	name := vc.GetName()
	commands := [][]string{
		{
			"--kubeconfig",
			HostClusterConfigPath(),
			"-n", namespace,
			"rollout",
			"restart",
			fmt.Sprintf("deployment.apps/%s-kube-controller-manager", name),
			fmt.Sprintf("deployment.apps/%s-virtualcluster-scheduler", name),
			fmt.Sprintf("deployment.apps/%s-coredns", name),
		},
	}

	for _, args := range commands {
		klog.InfoS("run command:", strings.Join(args, " "))
		if err := runKubectlCommand(args...); err != nil {
			klog.InfoS("run command failed:", err)
		}
	}

	// wait for pod ready
	return WaitPodReady(data.RemoteClient(), namespace)
}

func UpdateVCAdminConfigClusteripSecret(data *Option) error {
	vc := data.VirtualCluster()

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		clusterSecret, err := data.RemoteClient().CoreV1().Secrets(vc.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-admin-config-clusterip", vc.GetName()), metav1.GetOptions{})
		if err != nil {
			return err
		}

		kubeconfitstr := string(clusterSecret.Data[constants.KubeConfig])

		val, ok := data.tmpValueMap["client"].(int)
		if !ok {
			return fmt.Errorf("get client port from map failed")
		}
		apiserverPort := int32(val)

		oldApiserverPort := vc.Status.PortMap["apiserver-port"]

		kubeconfitstr = strings.ReplaceAll(kubeconfitstr, string(oldApiserverPort), string(apiserverPort))

		clusterSecret.Data[constants.KubeConfig] = []byte(kubeconfitstr)
		_, err = data.RemoteClient().CoreV1().Secrets(vc.GetNamespace()).Update(context.TODO(), clusterSecret, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		return err
	}

	// update VC
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currentVC, err := data.KosmosClient().KosmosV1alpha1().VirtualClusters(vc.GetNamespace()).Get(context.TODO(), vc.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		currentVC.Spec.KubeInKubeConfig.APIServerServiceType = "nodePort"
		_, err = data.KosmosClient().KosmosV1alpha1().VirtualClusters(vc.GetNamespace()).Update(context.TODO(), currentVC, metav1.UpdateOptions{})

		data.UpdateVirtualCluster(currentVC)
		return err
	})

	return err
}
