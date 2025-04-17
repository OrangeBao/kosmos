package cert

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var RenewCertExample = templates.Examples(i18n.T(`
     # Renew cert, e.g:
     kosmosctl renew cert --kubeconfig=xxxx  --namespace=xxxx --name=xxxx --agent-user=xxxx --agent-pass=xxxx
`))

type CertCmdOptions struct {
	CertOptions CertOptions
}

func NewCmdRenewCert() *cobra.Command {
	o := &CertCmdOptions{}
	cmd := &cobra.Command{
		Use:                   "cert",
		Short:                 i18n.T("renew cert for virtual cluster. "),
		Long:                  "",
		Example:               RenewCertExample,
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

func (o *CertCmdOptions) Complete() (err error) {
	return nil
}

func (o *CertCmdOptions) Validate() error {
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

func (o *CertCmdOptions) initEnv() {
	os.Setenv("KUBECONFIG", o.CertOptions.KubeconfigPath)
	os.Setenv("WEB_USER", o.CertOptions.WebUser)
	os.Setenv("WEB_PASS", o.CertOptions.WebPass)
}

func (o *CertCmdOptions) Run() error {
	r, err := NewCertOption(&o.CertOptions)
	o.initEnv()
	if err != nil {
		return err
	}

	err = RunTask([]TaskFunc{
		RunCheckEnvironment,
		RunBackupSecrets,
		RunReCreateCertAndKubeConfig,
		UpdateKubeProxyConfig,
		RestartVirtualControlPlanePod,
		RestartVirtualWorkerKubelet,
		RestartVirtualPod,
	}, r)
	if err != nil {
		return err
	}
	klog.Infof("############ renew cert success!!!!")
	return nil
}
