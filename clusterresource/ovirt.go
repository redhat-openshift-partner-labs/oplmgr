package clusterresource

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	installertypes "github.com/openshift/installer/pkg/types"
	installerovirt "github.com/openshift/installer/pkg/types/ovirt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivev1ovirt "github.com/openshift/hive/apis/hive/v1/ovirt"
	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
)

var _ CloudBuilder = (*OvirtCloudBuilder)(nil)

// OvirtCloudBuilder encapsulates cluster artifact generation logic specific to oVirt.
type OvirtCloudBuilder struct {
	// OvirtConfig is the data that will be used as the ovirt-config.yaml file for
	// cluster provisioning.
	OvirtConfig []byte
	// The target cluster under which all VMs will run
	ClusterID string
	// The target storage domain under which all VM disk would be created.
	StorageDomainID string
	// The target network of all the network interfaces of the nodes. Omitting defaults to ovirtmgmt
	// network which is a default network for every oVirt cluster.
	NetworkName string
	// APIVIP is an IP which will be served by bootstrap and then pivoted masters, using keepalived
	APIVIP string
	// IngressIP is an external IP which routes to the default ingress controller.
	// The IP is a suitable target of a wildcard DNS record used to resolve default route host names.
	IngressVIP string
	// CACert is the CA certificate(s) used to communicate with oVirt.
	CACert []byte
}

func (p *OvirtCloudBuilder) GenerateCredentialsSecret(o *Builder) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.CredsSecretName(o),
			Namespace: o.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			OvirtCredentialsName: p.OvirtConfig,
		},
	}
}

func (p *OvirtCloudBuilder) GenerateCloudObjects(o *Builder) []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.certificatesSecretName(o),
				Namespace: o.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				".cacert": p.CACert,
			},
		},
	}
}

func (p *OvirtCloudBuilder) GetCloudPlatform(o *Builder) hivev1.Platform {
	return hivev1.Platform{
		Ovirt: &hivev1ovirt.Platform{
			ClusterID: p.ClusterID,
			CredentialsSecretRef: corev1.LocalObjectReference{
				Name: p.CredsSecretName(o),
			},
			CertificatesSecretRef: corev1.LocalObjectReference{
				Name: p.certificatesSecretName(o),
			},
			StorageDomainID: p.StorageDomainID,
			NetworkName:     p.NetworkName,
		},
	}
}

func (p *OvirtCloudBuilder) addMachinePoolPlatform(o *Builder, mp *hivev1.MachinePool) {
	mp.Spec.Platform.Ovirt = &hivev1ovirt.MachinePool{
		CPU: &hivev1ovirt.CPU{
			Sockets: 1,
			Cores:   4,
		},
		MemoryMB: 16348,
		OSDisk: &hivev1ovirt.Disk{
			SizeGB: 120,
		},
		VMType: hivev1ovirt.VMTypeServer,
	}
}

func (p *OvirtCloudBuilder) addInstallConfigPlatform(o *Builder, ic *installertypes.InstallConfig) {
	ic.Platform = installertypes.Platform{
		Ovirt: &installerovirt.Platform{
			ClusterID:       p.ClusterID,
			StorageDomainID: p.StorageDomainID,
			NetworkName:     p.NetworkName,
			APIVIP:          p.APIVIP,
			IngressVIP:      p.IngressVIP,
		},
	}
}

func (p *OvirtCloudBuilder) CredsSecretName(o *Builder) string {
	return fmt.Sprintf("%s-ovirt-creds", o.Name)
}

func (p *OvirtCloudBuilder) certificatesSecretName(o *Builder) string {
	return fmt.Sprintf("%s-ovirt-certs", o.Name)
}
