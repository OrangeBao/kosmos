package options

import (
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	componentbaseconfig "k8s.io/component-base/config"

	"github.com/kosmos.io/kosmos/pkg/utils"
)

type Options struct {
	LeaderElection             componentbaseconfig.LeaderElectionConfiguration
	KubernetesOptions          KubernetesOptions
	KubeNestOptions            KubeNestOptions
	AllowNodeOwnbyMulticluster bool
	KosmosJoinController       bool
}

type KubernetesOptions struct {
	KubeConfig string
	Master     string
	QPS        float32
	Burst      int
}

type KubeNestOptions struct {
	ForceDestroy      bool
	AnpMode           string
	AdmissionPlugins  bool
	ApiServerReplicas int
}

func NewOptions() *Options {
	return &Options{
		LeaderElection: componentbaseconfig.LeaderElectionConfiguration{
			LeaderElect:       true,
			ResourceLock:      resourcelock.LeasesResourceLock,
			ResourceNamespace: utils.DefaultNamespace,
			ResourceName:      "virtual-cluster-controller",
		},
	}
}

func (o *Options) AddFlags(flags *pflag.FlagSet) {
	if o == nil {
		return
	}

	flags.BoolVar(&o.LeaderElection.LeaderElect, "leader-elect", true, "Start a leader election client and gain leadership before executing the main loop. Enable this when running replicated components for high availability.")
	flags.StringVar(&o.LeaderElection.ResourceName, "leader-elect-resource-name", "operator", "The name of resource object that is used for locking during leader election.")
	flags.StringVar(&o.LeaderElection.ResourceNamespace, "leader-elect-resource-namespace", utils.DefaultNamespace, "The namespace of resource object that is used for locking during leader election.")
	flags.Float32Var(&o.KubernetesOptions.QPS, "kube-qps", 40.0, "QPS to use while talking with kube-apiserver.")
	flags.IntVar(&o.KubernetesOptions.Burst, "kube-burst", 60, "Burst to use while talking with kube-apiserver.")
	flags.StringVar(&o.KubernetesOptions.KubeConfig, "kubeconfig", "", "Path for kubernetes kubeconfig file, if left blank, will use in cluster way.")
	flags.StringVar(&o.KubernetesOptions.Master, "master", "", "Used to generate kubeconfig for downloading, if not specified, will use host in kubeconfig.")
	flags.BoolVar(&o.AllowNodeOwnbyMulticluster, "multiowner", false, "Allow node own by multicluster or not.")
	flags.BoolVar(&o.KosmosJoinController, "kosmos-join-controller", false, "Turn on or off kosmos-join-controller.")
	flags.BoolVar(&o.KubeNestOptions.ForceDestroy, "kube-nest-force-destroy", false, "Force destroy the node.If it set true.If set to true, Kubernetes will not evict the existing nodes on the node when joining nodes to the tenant's control plane, but will instead force destroy.")
	flags.StringVar(&o.KubeNestOptions.AnpMode, "kube-nest-anp-mode", "tcp", "kube-apiserver network proxy mode, must be set to tcp or uds. uds mode the replicas for apiserver should be one, and tcp for multi apiserver replicas.")
	flags.BoolVar(&o.KubeNestOptions.AdmissionPlugins, "kube-nest-admission-plugins", false, "kube-apiserver network disable-admission-plugins, false for - --disable-admission-plugins=License, true for remove the --disable-admission-plugins=License flag .")
	flags.IntVar(&o.KubeNestOptions.ApiServerReplicas, "kube-nest-apiserver-replicas", 1, "virtual-cluster kube-apiserver replicas. default is 2.")
}
