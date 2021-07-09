package utils

import (
	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

// GetAWSCreds reads AWS credentials either from either the specified credentials file,
// the standard environment variables, or a default credentials file. (~/.aws/credentials)
// The defaultCredsFile will only be used if credsFile is empty and the environment variables
// are not set.
func GetAWSCreds(credsFile, defaultCredsFile string) (string, string, error) {
	credsFilePath := defaultCredsFile
	switch {
	case credsFile != "":
		credsFilePath = credsFile
	default:
		secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
		if len(secretAccessKey) > 0 && len(accessKeyID) > 0 {
			return accessKeyID, secretAccessKey, nil
		}
	}
	credFile, err := ini.Load(credsFilePath)
	if err != nil {
		log.Error("Cannot load AWS credentials")
		return "", "", err
	}
	defaultSection, err := credFile.GetSection("default")
	if err != nil {
		log.Error("Cannot get default section from AWS credentials file")
		return "", "", err
	}
	accessKeyIDValue := defaultSection.Key("aws_access_key_id")
	secretAccessKeyValue := defaultSection.Key("aws_secret_access_key")
	if accessKeyIDValue == nil || secretAccessKeyValue == nil {
		log.Error("AWS credentials file missing keys in default section")
	}
	return accessKeyIDValue.String(), secretAccessKeyValue.String(), nil
}

// GetAzureCreds reads Azure credentials used for install/uninstall from either the default
// credentials file (~/.azure/osServiceAccount.json), the standard environment variable,
// or provided credsFile location (in increasing order of preference).
func GetAzureCreds(credsFile string) ([]byte, error) {
	credsFilePath := filepath.Join(homedir.HomeDir(), ".azure", AzureCredentialsName)
	if l := os.Getenv(AzureCredentialsEnvVar); l != "" {
		credsFilePath = l
	}
	if credsFile != "" {
		credsFilePath = credsFile
	}
	log.WithField("credsFilePath", credsFilePath).Info("Loading azure creds")
	return ioutil.ReadFile(credsFilePath)
}

// GetGCPCreds reads GCP credentials either from either the specified credentials file,
// the standard environment variables, or a default credentials file. (~/.gcp/osServiceAccount.json)
// The defaultCredsFile will only be used if credsFile is empty and the environment variables
// are not set.
func GetGCPCreds(credsFile string) ([]byte, error) {
	credsFilePath := filepath.Join(homedir.HomeDir(), ".gcp", GCPCredentialsName)
	if l := os.Getenv("GOOGLE_CREDENTIALS"); l != "" {
		credsFilePath = l
	}
	if credsFile != "" {
		credsFilePath = credsFile
	}
	log.WithField("credsFilePath", credsFilePath).Info("Loading gcp service account")
	return ioutil.ReadFile(credsFilePath)
}

// GetOpenStackCreds reads OpenStack credentials either from the specified credentials file,
// ~/.config/openstack/clouds.yaml, or /etc/openstack/clouds.yaml
func GetOpenStackCreds(credsFile string) ([]byte, error) {
	if credsFile == "" {
		for _, filePath := range []string{filepath.Join(homedir.HomeDir(), ".config", "openstack", OpenStackCredentialsName),
			"/etc/openstack"} {

			_, err := os.Stat(filePath)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			if os.IsNotExist(err) {
				continue
			}
			credsFile = filePath
			break
		}
	}
	log.WithField("credsFile", credsFile).Info("Loading OpenStack creds")
	return ioutil.ReadFile(credsFile)
}

// GetOvirtCreds reads oVirt credentials either from the specified credentials file,
// or ~/.ovirt/ovirt-config.yaml
func GetOvirtCreds(credsFile string) ([]byte, error) {
	if credsFile == "" {
		for _, filePath := range []string{filepath.Join(homedir.HomeDir(), ".ovirt", OvirtCredentialsName)} {

			_, err := os.Stat(filePath)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			if os.IsNotExist(err) {
				continue
			}
			credsFile = filePath
			break
		}
	}
	return ioutil.ReadFile(credsFile)
}