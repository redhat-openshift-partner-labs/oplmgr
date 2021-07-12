/*
Copyright © 2021 Melvin Hillsman <mrhillsman@redhat.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/redhat-openshift-partner-labs/oplmgr/clusterresource"
	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
	. "github.com/redhat-openshift-partner-labs/oplmgr/utils"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	//"github.com/openshift/hive/contrib/pkg/utils"
	//awsutils "github.com/openshift/hive/contrib/pkg/utils/aws"
	//azurecredutil "github.com/openshift/hive/contrib/pkg/utils/azure"
	//gcputils "github.com/openshift/hive/contrib/pkg/utils/gcp"
	//openstackutils "github.com/openshift/hive/contrib/pkg/utils/openstack"
	//ovirtutils "github.com/openshift/hive/contrib/pkg/utils/ovirt"
)

// Options is the set of options to generate and apply a new cluster deployment
type Options struct {
	Name                              string
	Namespace                         string
	SSHPublicKeyFile                  string
	SSHPublicKey                      string
	SSHPrivateKeyFile                 string
	SSHPrivateKey                     string
	BaseDomain                        string
	PullSecret                        string
	PullSecretFile                    string
	BoundServiceAccountSigningKeyFile string
	Cloud                             string
	CredsFile                         string
	CredentialsModeManual             bool
	ClusterImageSet                   string
	ReleaseImage                      string
	ReleaseImageSource                string
	DeleteAfter                       string
	HibernateAfter                    string
	HibernateAfterDur                 *time.Duration
	ServingCert                       string
	ServingCertKey                    string
	UseClusterImageSet                bool
	ManageDNS                         bool
	Output                            string
	IncludeSecrets                    bool
	InstallOnce                       bool
	UninstallOnce                     bool
	SimulateBootstrapFailure          bool
	WorkerNodesCount                  int64
	CreateSampleSyncsets              bool
	ManifestsDir                      string
	Adopt                             bool
	AdoptAdminKubeConfig              string
	AdoptInfraID                      string
	AdoptClusterID                    string
	AdoptAdminUsername                string
	AdoptAdminPassword                string
	MachineNetwork                    string
	Region                            string
	Labels                            []string
	Annotations                       []string
	SkipMachinePools                  bool
	AdditionalTrustBundle             string
	CentralMachineManagement          bool
	Internal                          bool

	// AWS
	AWSUserTags    []string
	AWSPrivateLink bool

	// Azure
	AzureBaseDomainResourceGroupName string

	// OpenStack
	OpenStackCloud             string
	OpenStackExternalNetwork   string
	OpenStackMasterFlavor      string
	OpenStackComputeFlavor     string
	OpenStackAPIFloatingIP     string
	OpenStackIngressFloatingIP string

	// VSphere
	VSphereVCenter          string
	VSphereDatacenter       string
	VSphereDefaultDataStore string
	VSphereFolder           string
	VSphereCluster          string
	VSphereAPIVIP           string
	VSphereIngressVIP       string
	VSphereNetwork          string
	VSphereCACerts          string

	// Ovirt
	OvirtClusterID       string
	OvirtStorageDomainID string
	OvirtNetworkName     string
	OvirtAPIVIP          string
	OvirtIngressVIP      string
	OvirtCACerts         string

	homeDir string
	log     log.FieldLogger
}

var opt Options

const (
	cloudAWS             = "aws"
	cloudAzure           = "azure"
	cloudGCP             = "gcp"
	cloudOpenStack       = "openstack"
	cloudVSphere         = "vsphere"
	cloudOVirt           = "ovirt"
	cloudIBM             = "ibm"
)

var (
	validClouds = map[string]bool{
		cloudAWS:       true,
		cloudAzure:     true,
		cloudGCP:       true,
		cloudOpenStack: true,
		cloudVSphere:   true,
		cloudOVirt:     true,
		cloudIBM:       false,
	}
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Create a Hive ClusterDeployment",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := opt.Complete(cmd, args); err != nil {
			opt.log.WithError(err).Fatal("Error")
		}

		if err := opt.Validate(cmd); err != nil {
			opt.log.WithError(err).Fatal("Error")
		}

		if err := opt.Run(); err != nil {
			opt.log.WithError(err).Fatal("Error")
		}
	},
}

func init() {
	flags := provisionCmd.Flags()

	flags.StringVar(&opt.Cloud, "cloud", cloudAWS, "Cloud provider: aws|azure|gcp|openstack|ibm (currently ibm is unavailable)")
	flags.StringVarP(&opt.Namespace, "namespace", "n", "hive", "Namespace to create cluster deployment in")
	//flags.StringVar(&opt.SSHPrivateKeyFile, "ssh-private-key-file", "defaultSSHPrivateKeyFile", "file name of SSH private key for cluster")
	//flags.StringVar(&opt.SSHPrivateKey, "ssh-private-key", "", "SSH private key for cluster")
	//flags.StringVar(&opt.SSHPublicKeyFile, "ssh-public-key-file", defaultSSHPublicKeyFile, "file name of SSH public key for cluster")
	//flags.StringVar(&opt.SSHPublicKey, "ssh-public-key", "", "SSH public key for cluster")
	flags.StringVar(&opt.BaseDomain, "base-domain", "partner-lab.rhecoeng.com", "Base domain for the cluster")
	flags.StringVar(&opt.PullSecret, "pull-secret", "", "Pull secret for cluster. Takes precedence over pull-secret-file.")
	flags.StringVar(&opt.DeleteAfter, "delete-after", "", "Delete this cluster after the given duration. (e.g. 8h)")
	flags.StringVar(&opt.HibernateAfter, "hibernate-after", "", "Automatically hibernate the cluster whenever it has been running for the given duration")
	//flags.StringVar(&opt.PullSecretFile, "pull-secret-file", defaultPullSecretFile, "file name of pull secret for cluster")
	flags.StringVar(&opt.BoundServiceAccountSigningKeyFile, "bound-service-account-signing-key-file", "", "Private service account signing key (often created with ccoutil create key-pair)")
	flags.BoolVar(&opt.CredentialsModeManual, "credentials-mode-manual", false, "Configure the Cloud Credential Operator in the target cluster to Manual mode. Implies the use of --manifests-dir to inject custom Secrets for all CredentialsRequests in the cluster.")

	flags.StringVar(&opt.CredsFile, "creds-file", "", "Cloud credentials file (defaults vary depending on cloud)")
	flags.StringVar(&opt.ClusterImageSet, "image-set", "", "Cluster image set to use for this cluster deployment")
	flags.StringVar(&opt.ReleaseImage, "release-image", "", "Release image to use for installing this cluster deployment")
	flags.StringVar(&opt.ReleaseImageSource, "release-image-source", "https://amd64.ocp.releases.ci.openshift.org/api/v1/releasestream/4-stable/latest", "URL to JSON describing the release image pull spec")
	flags.StringVar(&opt.ServingCert, "serving-cert", "", "Serving certificate for control plane and routes")
	flags.StringVar(&opt.ServingCertKey, "serving-cert-key", "", "Serving certificate key for control plane and routes")
	flags.BoolVar(&opt.ManageDNS, "manage-dns", false, "Manage this cluster's DNS. This is only available for AWS and GCP.")
	flags.BoolVar(&opt.UseClusterImageSet, "use-image-set", true, "If true, use a cluster image set for this cluster")
	flags.StringVarP(&opt.Output, "output", "o", "", "Output of this command (nothing will be created on cluster). Valid values: yaml,json")
	flags.BoolVar(&opt.IncludeSecrets, "include-secrets", true, "Include secrets along with ClusterDeployment")
	flags.BoolVar(&opt.InstallOnce, "install-once", false, "Run the install only one time and fail if not successful")
	flags.BoolVar(&opt.UninstallOnce, "uninstall-once", false, "Run the uninstall only one time and fail if not successful")
	//flags.BoolVar(&opt.SimulateBootstrapFailure, "simulate-bootstrap-failure", false, "Simulate an install bootstrap failure by injecting an invalid manifest.")
	flags.Int64Var(&opt.WorkerNodesCount, "workers", 3, "Number of worker nodes to create.")
	//flags.BoolVar(&opt.CreateSampleSyncsets, "create-sample-syncsets", false, "Create a set of sample syncsets for testing")
	flags.StringVar(&opt.ManifestsDir, "manifests", "", "Directory containing manifests to add during installation")
	flags.StringVar(&opt.MachineNetwork, "machine-network", "10.0.0.0/16", "Cluster's MachineNetwork to pass to the installer")
	flags.StringVar(&opt.Region, "region", "", "Region to which to install the cluster. This is only relevant to AWS, Azure, and GCP.")
	flags.StringSliceVarP(&opt.Labels, "labels", "l", nil, "Label to apply to the ClusterDeployment (key=val)")
	flags.StringSliceVarP(&opt.Annotations, "annotations", "a", nil, "Annotation to apply to the ClusterDeployment (key=val)")
	flags.BoolVar(&opt.SkipMachinePools, "skip-machine-pools", false, "Skip generation of Hive MachinePools for day 2 MachineSet management")
	flags.BoolVar(&opt.CentralMachineManagement, "central-machine-mgmt", false, "Enable central machine management for cluster")
	flags.BoolVar(&opt.Internal, "internal", false, `When set, it configures the install-config.yaml's publish field to Internal.
OpenShift Installer publishes all the services of the cluster like API server and ingress to internal network and not the Internet.`)

	// Flags related to adoption.
	flags.BoolVar(&opt.Adopt, "adopt", false, "Enable adoption mode for importing a pre-existing cluster into Hive. Will require additional flags for adoption info.")
	flags.StringVar(&opt.AdoptAdminKubeConfig, "adopt-admin-kubeconfig", "", "Path to a cluster admin kubeconfig file for a cluster being adopted. (required if using --adopt)")
	flags.StringVar(&opt.AdoptInfraID, "adopt-infra-id", "", "Infrastructure ID for this cluster's cloud provider. (required if using --adopt)")
	flags.StringVar(&opt.AdoptClusterID, "adopt-cluster-id", "", "Cluster UUID used for telemetry. (required if using --adopt)")
	flags.StringVar(&opt.AdoptAdminUsername, "adopt-admin-username", "", "Username for cluster web console administrator. (optional)")
	flags.StringVar(&opt.AdoptAdminPassword, "adopt-admin-password", "", "Password for cluster web console administrator. (optional)")

	// AWS flags
	flags.StringSliceVar(&opt.AWSUserTags, "aws-user-tags", nil, "Additional tags to add to resources. Must be in the form \"key=value\"")
	flags.BoolVar(&opt.AWSPrivateLink, "aws-private-link", false, "Enables access to cluster using AWS PrivateLink")

	// Azure flags
	flags.StringVar(&opt.AzureBaseDomainResourceGroupName, "azure-base-domain-resource-group-name", "os4-common", "Resource group where the azure DNS zone for the base domain is found")

	// OpenStack flags
	flags.StringVar(&opt.OpenStackCloud, "openstack-cloud", "openstack", "Section of clouds.yaml to use for API/auth")
	flags.StringVar(&opt.OpenStackExternalNetwork, "openstack-external-network", "provider_net_shared_3", "External OpenStack network name to deploy into")
	flags.StringVar(&opt.OpenStackMasterFlavor, "openstack-master-flavor", "ci.m4.xlarge", "Compute flavor to use for master nodes")
	flags.StringVar(&opt.OpenStackComputeFlavor, "openstack-compute-flavor", "m1.large", "Compute flavor to use for worker nodes")
	flags.StringVar(&opt.OpenStackAPIFloatingIP, "openstack-api-floating-ip", "", "Floating IP address to use for cluster's API")
	flags.StringVar(&opt.OpenStackIngressFloatingIP, "openstack-ingress-floating-ip", "", "Floating IP address to use for cluster's Ingress service")

	// vSphere flags
	flags.StringVar(&opt.VSphereVCenter, "vsphere-vcenter", "", "Domain name or IP address of the vCenter")
	flags.StringVar(&opt.VSphereDatacenter, "vsphere-datacenter", "", "Datacenter to use in the vCenter")
	flags.StringVar(&opt.VSphereDefaultDataStore, "vsphere-default-datastore", "", "Default datastore to use for provisioning volumes")
	flags.StringVar(&opt.VSphereFolder, "vsphere-folder", "", "Folder that will be used and/or created for virtual machines")
	flags.StringVar(&opt.VSphereCluster, "vsphere-cluster", "", "Cluster virtual machines will be cloned into")
	flags.StringVar(&opt.VSphereAPIVIP, "vsphere-api-vip", "", "Virtual IP address for the api endpoint")
	flags.StringVar(&opt.VSphereIngressVIP, "vsphere-ingress-vip", "", "Virtual IP address for ingress application routing")
	flags.StringVar(&opt.VSphereNetwork, "vsphere-network", "", "Name of the network to be used by the cluster")
	flags.StringVar(&opt.VSphereCACerts, "vsphere-ca-certs", "", "Path to vSphere CA certificate, multiple CA paths can be : delimited")

	// oVirt flags
	flags.StringVar(&opt.OvirtClusterID, "ovirt-cluster-id", "", "The oVirt cluster id (uuid) under which all VMs will run")
	flags.StringVar(&opt.OvirtStorageDomainID, "ovirt-storage-domain-id", "", "oVirt storage domain id (uuid) under which all VM disk would be created")
	flags.StringVar(&opt.OvirtNetworkName, "ovirt-network-name", "ovirtmgmt", "oVirt network name")
	flags.StringVar(&opt.OvirtAPIVIP, "ovirt-api-vip", "", "IP which will be served by bootstrap and then pivoted masters, using keepalived")
	flags.StringVar(&opt.OvirtIngressVIP, "ovirt-ingress-vip", "", "External IP which routes to the default ingress controller")
	flags.StringVar(&opt.OvirtCACerts, "ovirt-ca-certs", "", "Path to oVirt CA certificate, multiple CA paths can be : delimited")

	// Additional CA Trust Bundle
	flags.StringVar(&opt.AdditionalTrustBundle, "additional-trust-bundle", "", "Path to a CA Trust Bundle which will be added to the nodes trusted certificate store.")

	rootCmd.AddCommand(provisionCmd)
}

// Complete finishes parsing arguments for the command
func (o *Options) Complete(cmd *cobra.Command, args []string) error {
	uuid, err := cmd.Flags().GetString("cluster-id")
	if err != nil {
		log.Printf("Unable to get the cluster-id: %v\n", err)
	}

	company, err := cmd.Flags().GetString("company")
	if err != nil {
		log.Printf("Unable to get the company: %v\n", err)
	}

	re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	company = re.ReplaceAllString(company, "")
	company = strings.TrimSpace(strings.ToLower(company))

	o.Name = strings.Split(uuid, "-")[0] + "-" + company

	if o.Region == "" {
		switch o.Cloud {
		case cloudAWS:
			o.Region = "us-west-2"
		case cloudAzure:
			o.Region = "centralus"
		case cloudGCP:
			o.Region = "us-east1"
		}
	}

	if o.HibernateAfter != "" {
		dur, err := time.ParseDuration(o.HibernateAfter)
		if err != nil {
			return errors.Wrapf(err, "unable to parse HibernateAfter duration")
		}
		o.HibernateAfterDur = &dur
	}

	publickey, privatekey := GenerateSSHKeys()
	o.SSHPublicKey = string(publickey)
	o.SSHPrivateKey = string(privatekey)

	return nil
}

// Validate ensures that option values make sense
func (o *Options) Validate(cmd *cobra.Command) error {
	if len(o.Output) > 0 && o.Output != "yaml" && o.Output != "json" {
		cmd.Usage()
		o.log.Info("Invalid value for output. Valid values are: yaml, json.")
		return fmt.Errorf("invalid output")
	}
	if !o.UseClusterImageSet && len(o.ClusterImageSet) > 0 {
		cmd.Usage()
		o.log.Info("If not using cluster image sets, do not specify the name of one")
		return fmt.Errorf("invalid option")
	}
	if len(o.ServingCert) > 0 && len(o.ServingCertKey) == 0 {
		cmd.Usage()
		o.log.Info("If specifying a serving certificate, specify a valid serving certificate key")
		return fmt.Errorf("invalid serving cert")
	}
	if !validClouds[o.Cloud] {
		cmd.Usage()
		o.log.Infof("Unsupported cloud: %s", o.Cloud)
		return fmt.Errorf("unsupported cloud: %s", o.Cloud)
	}
	if o.Cloud == cloudOpenStack {
		if o.OpenStackAPIFloatingIP == "" {
			o.log.Info("Missing openstack-api-floating-ip parameter")
			return fmt.Errorf("missing openstack-api-floating-ip parameter")
		}
		if o.OpenStackCloud == "" {
			o.log.Info("Missing openstack-cloud parameter")
			return fmt.Errorf("missing openstack-cloud parameter")
		}
	}

	if o.CredentialsModeManual {
		if o.ManifestsDir == "" {
			return fmt.Errorf("--credentials-mode-manual requires --manifests-dir containing custom Secrets with manually provisioned credentials")
		}
	}

	if o.AWSPrivateLink && o.Cloud != cloudAWS {
		return fmt.Errorf("--aws-private-link can only be enabled for AWS cloud platform")
	}

	if o.Adopt {
		if o.AdoptAdminKubeConfig == "" || o.AdoptInfraID == "" || o.AdoptClusterID == "" {
			return fmt.Errorf("must specify the following options when using --adopt: --adopt-admin-kube-config, --adopt-infra-id, --adopt-cluster-id")
		}

		if _, err := os.Stat(o.AdoptAdminKubeConfig); os.IsNotExist(err) {
			return fmt.Errorf("--adopt-admin-kubeconfig does not exist: %s", o.AdoptAdminKubeConfig)
		}

		// Admin username and password must both be specified if either are.
		if (o.AdoptAdminUsername != "" || o.AdoptAdminPassword != "") && !(o.AdoptAdminUsername != "" && o.AdoptAdminPassword != "") {
			return fmt.Errorf("--adopt-admin-username and --adopt-admin-password must be used together")
		}
	} else {
		if o.AdoptAdminKubeConfig != "" || o.AdoptInfraID != "" || o.AdoptClusterID != "" || o.AdoptAdminUsername != "" || o.AdoptAdminPassword != "" {
			return fmt.Errorf("cannot use adoption options without --adopt: --adopt-admin-kube-config, --adopt-infra-id, --adopt-cluster-id, --adopt-admin-username, --adopt-admin-password")
		}
	}

	if o.Region != "" {
		switch c := o.Cloud; c {
		case cloudAWS, cloudAzure, cloudGCP:
		default:
			return fmt.Errorf("cannot specify region when cloud is %q", c)
		}
	}

	for _, ls := range o.Labels {
		tokens := strings.Split(ls, "=")
		if len(tokens) != 2 {
			return fmt.Errorf("unable to parse key=value label: %s", ls)
		}
	}
	for _, ls := range o.Annotations {
		tokens := strings.Split(ls, "=")
		if len(tokens) != 2 {
			return fmt.Errorf("unable to parse key=value annotation: %s", ls)
		}
	}
	return nil
}

// Run executes the command
func (o *Options) Run() error {
	if err := hivev1.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	objs, err := o.GenerateObjects()
	if err != nil {
		return err
	}
	if len(o.Output) > 0 {
		var printer printers.ResourcePrinter
		if o.Output == "yaml" {
			printer = &printers.YAMLPrinter{}
		} else {
			printer = &printers.JSONPrinter{}
		}
		printObjects(objs, scheme.Scheme, printer)
		return err
	}
	rh, err := GetResourceHelper(o.log)
	if err != nil {
		return err
	}
	if len(o.Namespace) == 0 {
		o.Namespace, err = DefaultNamespace()
		if err != nil {
			o.log.Error("Cannot determine default namespace")
			return err
		}
	}
	for _, obj := range objs {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			o.log.WithError(err).Errorf("Cannot create accessor for object of type %T", obj)
			return err
		}
		accessor.SetNamespace(o.Namespace)
		if _, err := rh.ApplyRuntimeObject(obj, scheme.Scheme); err != nil {
			return err
		}

	}
	return nil
}

// GenerateObjects generates resources for a new cluster deployment
func (o *Options) GenerateObjects() ([]runtime.Object, error) {

	pullSecret, err := GetPullSecret(o.log, o.PullSecret, o.PullSecretFile)
	if err != nil {
		return nil, err
	}

	sshPrivateKey, err := o.getSSHPrivateKey()
	if err != nil {
		return nil, err
	}

	sshPublicKey, err := o.getSSHPublicKey()
	if err != nil {
		return nil, err
	}

	// Load installer manifest files:
	manifestFileData, err := o.getManifestFileBytes()
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)

	for _, ls := range o.Labels {
		tokens := strings.Split(ls, "=")
		labels[tokens[0]] = tokens[1]
	}

	annotations := map[string]string{}
	for _, ls := range o.Annotations {
		tokens := strings.Split(ls, "=")
		annotations[tokens[0]] = tokens[1]
	}

	builder := &clusterresource.Builder{
		Name:                     o.Name,
		Namespace:                o.Namespace,
		WorkerNodesCount:         o.WorkerNodesCount,
		PullSecret:               pullSecret,
		SSHPrivateKey:            sshPrivateKey,
		SSHPublicKey:             sshPublicKey,
		InstallOnce:              o.InstallOnce,
		BaseDomain:               o.BaseDomain,
		ManageDNS:                o.ManageDNS,
		DeleteAfter:              o.DeleteAfter,
		HibernateAfter:           o.HibernateAfterDur,
		Labels:                   labels,
		Annotations:              annotations,
		InstallerManifests:       manifestFileData,
		MachineNetwork:           o.MachineNetwork,
		SkipMachinePools:         o.SkipMachinePools,
		CentralMachineManagement: o.CentralMachineManagement,
	}
	if o.Adopt {
		kubeconfigBytes, err := ioutil.ReadFile(o.AdoptAdminKubeConfig)
		if err != nil {
			return nil, err
		}
		builder.Adopt = o.Adopt
		builder.AdoptInfraID = o.AdoptInfraID
		builder.AdoptClusterID = o.AdoptClusterID
		builder.AdoptAdminKubeconfig = kubeconfigBytes
		builder.AdoptAdminUsername = o.AdoptAdminUsername
		builder.AdoptAdminPassword = o.AdoptAdminPassword
	}
	if len(o.BoundServiceAccountSigningKeyFile) != 0 {
		signingKey, err := ioutil.ReadFile(o.BoundServiceAccountSigningKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", o.BoundServiceAccountSigningKeyFile, err)
		}
		builder.BoundServiceAccountSigningKey = string(signingKey)
	}

	switch o.Cloud {
	case cloudAWS:
		defaultCredsFilePath := filepath.Join(o.homeDir, ".aws", "credentials")
		accessKeyID, secretAccessKey, err := GetAWSCreds(o.CredsFile, defaultCredsFilePath)
		if err != nil {
			return nil, err
		}
		userTags := make(map[string]string, len(o.AWSUserTags))
		for _, t := range o.AWSUserTags {
			tagParts := strings.SplitN(t, "=", 2)
			switch len(tagParts) {
			case 0:
			case 1:
				userTags[tagParts[0]] = ""
			case 2:
				userTags[tagParts[0]] = tagParts[1]
			}
		}
		awsProvider := &clusterresource.AWSCloudBuilder{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			UserTags:        userTags,
			Region:          o.Region,
			PrivateLink:     o.AWSPrivateLink,
		}
		builder.CloudBuilder = awsProvider
	case cloudAzure:
		creds, err := GetAzureCreds(o.CredsFile)
		if err != nil {
			o.log.WithError(err).Error("Failed to read in Azure credentials")
			return nil, err
		}

		azureProvider := &clusterresource.AzureCloudBuilder{
			ServicePrincipal:            creds,
			BaseDomainResourceGroupName: o.AzureBaseDomainResourceGroupName,
			Region:                      o.Region,
		}
		builder.CloudBuilder = azureProvider
	case cloudGCP:
		creds, err := GetGCPCreds(o.CredsFile)
		if err != nil {
			return nil, err
		}
		projectID, err := ProjectID(creds)
		if err != nil {
			return nil, err
		}

		gcpProvider := &clusterresource.GCPCloudBuilder{
			ProjectID:      projectID,
			ServiceAccount: creds,
			Region:         o.Region,
		}
		builder.CloudBuilder = gcpProvider
	case cloudOpenStack:
		cloudsYAMLContent, err := GetOpenStackCreds(o.CredsFile)
		if err != nil {
			return nil, err
		}
		openStackProvider := &clusterresource.OpenStackCloudBuilder{
			Cloud:             o.OpenStackCloud,
			CloudsYAMLContent: cloudsYAMLContent,
			ExternalNetwork:   o.OpenStackExternalNetwork,
			ComputeFlavor:     o.OpenStackComputeFlavor,
			MasterFlavor:      o.OpenStackMasterFlavor,
			APIFloatingIP:     o.OpenStackAPIFloatingIP,
			IngressFloatingIP: o.OpenStackIngressFloatingIP,
		}
		builder.CloudBuilder = openStackProvider
	case cloudVSphere:
		vsphereUsername := os.Getenv(VSphereUsernameEnvVar)
		if vsphereUsername == "" {
			return nil, fmt.Errorf("no %s env var set, cannot proceed", VSphereUsernameEnvVar)
		}

		vspherePassword := os.Getenv(VSpherePasswordEnvVar)
		if vspherePassword == "" {
			return nil, fmt.Errorf("no %s env var set, cannot proceed", VSpherePasswordEnvVar)
		}

		vsphereCACerts := os.Getenv(VSphereTLSCACertsEnvVar)
		if o.VSphereCACerts != "" {
			vsphereCACerts = o.VSphereCACerts
		}
		if vsphereCACerts == "" {
			return nil, fmt.Errorf("must provide --vsphere-ca-certs or set %s env var set", VSphereTLSCACertsEnvVar)
		}
		var caCerts [][]byte
		for _, cert := range filepath.SplitList(vsphereCACerts) {
			caCert, err := ioutil.ReadFile(cert)
			if err != nil {
				return nil, fmt.Errorf("error reading %s: %w", cert, err)
			}
			caCerts = append(caCerts, caCert)
		}

		vSphereNetwork := os.Getenv(VSphereNetworkEnvVar)
		if o.VSphereNetwork != "" {
			vSphereNetwork = o.VSphereNetwork
		}

		vSphereDatacenter := os.Getenv(VSphereDataCenterEnvVar)
		if o.VSphereDatacenter != "" {
			vSphereDatacenter = o.VSphereDatacenter
		}
		if vSphereDatacenter == "" {
			return nil, fmt.Errorf("must provide --vsphere-datacenter or set %s env var", VSphereDataCenterEnvVar)
		}

		vSphereDatastore := os.Getenv(VSphereDataStoreEnvVar)
		if o.VSphereDefaultDataStore != "" {
			vSphereDatastore = o.VSphereDefaultDataStore
		}
		if vSphereDatastore == "" {
			return nil, fmt.Errorf("must provide --vsphere-default-datastore or set %s env var", VSphereDataStoreEnvVar)
		}

		vSphereVCenter := os.Getenv(VSphereVCenterEnvVar)
		if o.VSphereVCenter != "" {
			vSphereVCenter = o.VSphereVCenter
		}
		if vSphereVCenter == "" {
			return nil, fmt.Errorf("must provide --vsphere-vcenter or set %s env var", VSphereVCenterEnvVar)
		}

		vsphereProvider := &clusterresource.VSphereCloudBuilder{
			VCenter:          vSphereVCenter,
			Username:         vsphereUsername,
			Password:         vspherePassword,
			Datacenter:       vSphereDatacenter,
			DefaultDatastore: vSphereDatastore,
			Folder:           o.VSphereFolder,
			Cluster:          o.VSphereCluster,
			APIVIP:           o.VSphereAPIVIP,
			IngressVIP:       o.VSphereIngressVIP,
			Network:          vSphereNetwork,
			CACert:           bytes.Join(caCerts, []byte("\n")),
		}
		builder.CloudBuilder = vsphereProvider
	case cloudOVirt:
		oVirtConfig, err := GetOvirtCreds(o.CredsFile)
		if err != nil {
			return nil, err
		}
		if o.OvirtCACerts == "" {
			return nil, errors.New("must provide --ovirt-ca-certs")
		}
		var caCerts [][]byte
		for _, cert := range filepath.SplitList(o.OvirtCACerts) {
			caCert, err := ioutil.ReadFile(cert)
			if err != nil {
				return nil, fmt.Errorf("error reading %s: %w", cert, err)
			}
			caCerts = append(caCerts, caCert)
		}
		oVirtProvider := &clusterresource.OvirtCloudBuilder{
			OvirtConfig:     oVirtConfig,
			ClusterID:       o.OvirtClusterID,
			StorageDomainID: o.OvirtStorageDomainID,
			NetworkName:     o.OvirtNetworkName,
			APIVIP:          o.OvirtAPIVIP,
			IngressVIP:      o.OvirtIngressVIP,
			CACert:          bytes.Join(caCerts, []byte("\n")),
		}
		builder.CloudBuilder = oVirtProvider
		builder.SkipMachinePools = true
	}

	if o.Internal {
		builder.PublishStrategy = "Internal"
	}

	if len(o.ServingCert) != 0 {
		servingCert, err := ioutil.ReadFile(o.ServingCert)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", o.ServingCert, err)
		}
		builder.ServingCert = string(servingCert)
		servingCertKey, err := ioutil.ReadFile(o.ServingCertKey)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", o.ServingCertKey, err)
		}
		builder.ServingCertKey = string(servingCertKey)
	}

	imageSet, err := o.configureImages(builder)
	if err != nil {
		return nil, err
	}

	result, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// Add some additional objects we don't yet want to move to the cluster builder library.
	if imageSet != nil {
		result = append(result, imageSet)
	}

	return result, nil
}

func (o *Options) getSSHPublicKey() (string, error) {
	sshPublicKey := os.Getenv("PUBLIC_SSH_KEY")
	if len(sshPublicKey) > 0 {
		return sshPublicKey, nil
	}
	if len(o.SSHPublicKey) > 0 {
		return o.SSHPublicKey, nil
	}
	if len(o.SSHPublicKeyFile) > 0 {
		data, err := ioutil.ReadFile(o.SSHPublicKeyFile)
		if err != nil {
			o.log.Error("Cannot read SSH public key file")
			return "", err
		}
		sshPublicKey = strings.TrimSpace(string(data))
		return sshPublicKey, nil
	}

	o.log.Error("Cannot determine SSH key to use")
	return "", nil
}

func (o *Options) getSSHPrivateKey() (string, error) {
	if len(o.SSHPrivateKeyFile) > 0 {
		data, err := ioutil.ReadFile(o.SSHPrivateKeyFile)
		if err != nil {
			o.log.Error("Cannot read SSH private key file")
			return "", err
		}
		sshPrivateKey := strings.TrimSpace(string(data))
		return sshPrivateKey, nil
	}
	o.log.Debug("No private SSH key file provided")
	return "", nil
}

func (o *Options) getManifestFileBytes() (map[string][]byte, error) {
	if o.ManifestsDir == "" && !o.SimulateBootstrapFailure {
		return nil, nil
	}
	fileData := map[string][]byte{}
	if o.ManifestsDir != "" {
		files, err := ioutil.ReadDir(o.ManifestsDir)
		if err != nil {
			return nil, errors.Wrap(err, "could not read manifests directory")
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			data, err := ioutil.ReadFile(filepath.Join(o.ManifestsDir, file.Name()))
			if err != nil {
				return nil, errors.Wrapf(err, "could not read manifest file %q", file.Name())
			}
			fileData[file.Name()] = data
		}
	}
	return fileData, nil
}

func (o *Options) configureImages(generator *clusterresource.Builder) (*hivev1.ClusterImageSet, error) {
	if len(o.ClusterImageSet) > 0 {
		generator.ImageSet = o.ClusterImageSet
		return nil, nil
	}
	// TODO: move release image lookup code to the cluster library
	if o.ReleaseImage == "" {
		if o.ReleaseImageSource == "" {
			return nil, fmt.Errorf("specify either a release image or a release image source")
		}
		var err error
		o.ReleaseImage, err = DetermineReleaseImageFromSource(o.ReleaseImageSource)
		if err != nil {
			return nil, fmt.Errorf("cannot determine release image: %v", err)
		}
	}
	if !o.UseClusterImageSet {
		generator.ReleaseImage = o.ReleaseImage
		return nil, nil
	}

	imageSet := &hivev1.ClusterImageSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-imageset", o.Name),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterImageSet",
			APIVersion: hivev1.SchemeGroupVersion.String(),
		},
		Spec: hivev1.ClusterImageSetSpec{
			ReleaseImage: o.ReleaseImage,
		},
	}
	generator.ImageSet = imageSet.Name
	return imageSet, nil
}

func printObjects(objects []runtime.Object, scheme *runtime.Scheme, printer printers.ResourcePrinter) {
	typeSetterPrinter := printers.NewTypeSetter(scheme).ToPrinter(printer)
	switch len(objects) {
	case 0:
		return
	case 1:
		typeSetterPrinter.PrintObj(objects[0], os.Stdout)
	default:
		list := &metav1.List{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ListMeta: metav1.ListMeta{},
		}
		meta.SetList(list, objects)
		typeSetterPrinter.PrintObj(list, os.Stdout)
	}
}
