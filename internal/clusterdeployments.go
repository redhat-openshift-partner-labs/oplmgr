package internal

import (
	"context"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/aws"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterDeployments() map[string]interface{} {
	cfg, err := DefaultClientK8sAuthenticate()
	if err != nil {
		log.Printf("Unable to create default client: %v\n", err)
	}

	scheme := runtime.NewScheme()
	err = hivev1.AddToScheme(scheme)
	if err != nil {
		log.Printf("Unable to add hive to scheme: %v\n", err)
	}

	dc, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Printf("Unable to create K8s client: %v\n", err)
	}

	cdList := hivev1.ClusterDeploymentList{}

	err = dc.List(context.Background(), &cdList, &client.ListOptions{Namespace: "hive"})
	if err != nil {
		log.Printf("Unable to list ClusterDeployments: %v\n", err)
	}

	clusterDeployments := make(map[string]interface{})

	for _, cd := range cdList.Items {
		var details []string

		labels := make(map[string]string)
		for key, value := range cd.Labels {
			labels[key] = value
		}

		details = append(details,
			cd.Name,
			cd.Status.WebConsoleURL,
			cd.Spec.ClusterMetadata.AdminPasswordSecretRef.Name,
			cd.Spec.ClusterMetadata.AdminKubeconfigSecretRef.Name)

		clusterDeployments[cd.Spec.ClusterName] = map[string]interface{}{
			"details": details,
			"labels":  labels,
		}
	}

	return clusterDeployments
}

func CreateClusterDeployment(labRequest *LabRequest) {
	cfg, err := DefaultClientK8sAuthenticate()
	if err != nil {
		log.Printf("Unable to create default client: %v\n", err)
	}

	scheme := runtime.NewScheme()
	err = hivev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		log.Printf("Unable to add hive to scheme: %v\n", err)
	}

	dc, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Printf("Unable to create K8s client: %v\n", err)
	}

	kc := K8sAuthenticate()
	labSecret, err := kc.CoreV1().Secrets("hive").Get(context.Background(), labRequest.ID.String(), metav1.GetOptions{})
	if err != nil {
		log.Printf("Unable to get the lab secret; does it exist: %v\n", err)
	}

	// TODO: #1 Allow selection of platform; will require some Google Form changes and potentially capturing
	// information from partner specific to the platform cluster should be installed on
	// using AWS for now
	plat := hivev1.Platform{
		AWS: &aws.Platform{
			CredentialsSecretRef: corev1.LocalObjectReference{Name: "hive-aws-creds"},
			Region:               "us-west-2", // It could be useful to allow region selection based on where partner is
			UserTags:             map[string]string{"LabID": labRequest.ID.String()},
		},
	}

	secretRef := corev1.LocalObjectReference{Name: labRequest.ID.String()}

	leasetime := []string{"one-day", "one-week", "two-weeks", "one-month"}

	oplLabels := map[string]string{
		"opl-region":     labRequest.Availability,
		"opl-lease-time": leasetime[labRequest.LeaseTime],
	}

	charsFromID := strings.Split(labRequest.ID.String(), "-")[0]
	cds := hivev1.ClusterDeploymentSpec{
		ClusterName: labRequest.ClusterName + "-" + charsFromID,
		BaseDomain:  "opdev.io",
		Platform:    plat,
		ManageDNS:   false,
		Provisioning: &hivev1.Provisioning{
			InstallConfigSecretRef: &secretRef,
			ImageSetRef:            &hivev1.ClusterImageSetReference{Name: string(labSecret.Data["openshift"])},
			SSHPrivateKeySecretRef: &secretRef,
		},
	}

	cd := hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labRequest.ID.String(),
			Namespace: "hive",
			Labels:    oplLabels,
		},
		Spec: cds,
	}

	err = dc.Create(context.Background(), &cd)
	if err != nil {
		log.Printf("Unable to create cluster deployment: %v\n", err)
	}
}
