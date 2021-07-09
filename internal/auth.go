package internal

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/v33/github"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	"k8s.io/apimachinery/pkg/runtime"
	. "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimec "sigs.k8s.io/controller-runtime/pkg/client"
)

func GithubAuthenticate() (*github.Client, context.Context) {
	accesstoken := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accesstoken},
	)
	tc := oauth2.NewClient(ctx, ts)
	gc := github.NewClient(tc)
	return gc, ctx
}

func K8sAuthenticate() *kubernetes.Clientset {
	// create k8s client
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Printf("Unable to build config from flags: %v\n", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)

	return clientset
}

func HiveClientK8sAuthenticate() runtimec.Client {
	// create hive client
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Printf("Unable to build config from flags: %v\n", err)
	}

	nrs := runtime.NewScheme()
	err = hivev1.AddToScheme(nrs)
	if err != nil {
		log.Printf("Unable to add Hive scheme to client: %v\n", err)
	}

	hiveclient, err := runtimec.New(cfg, client.Options{Scheme: nrs})

	return hiveclient
}

func DefaultClientK8sAuthenticate() (*rest.Config, error) {
	cfg, err := clientcmd.LoadFromFile(os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Printf("The kubeconfig could not be loaded: %v\n", err)
	}
	dc := clientcmd.NewDefaultClientConfig(*cfg, &clientcmd.ConfigOverrides{})

	return dc.ClientConfig()
}

func DynamicClientK8sAuthenticate() (Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Printf("The kubeconfig could not be loaded: %v\n", err)
	}
	dc, err := NewForConfig(cfg)

	return dc, err
}

func GoogleDriveAuthenticate(credentials string, token string) (client *http.Client, err error) {
	credentialsFileBytes, err := ioutil.ReadFile(credentials)
	if err != nil {
		log.Printf("Unable to read credentials file: %v\n", err)
	}

	credentialsConfig, err := google.ConfigFromJSON(credentialsFileBytes,
		drive.DriveScope, drive.DriveFileScope, sheets.DriveScope, sheets.SpreadsheetsScope)
	if err != nil {
		log.Printf("Unable to create config from credentials file: %v\n", err)
	}

	tokenFile, err := os.Open(token)
	if err != nil {
		log.Printf("Unable to open token file: %v\n", err)
	}
	defer func(tokenFile *os.File) {
		err := tokenFile.Close()
		if err != nil {
			log.Printf("Unable to close token file: %v\n", err)
		}
	}(tokenFile)

	tokenJSON := &oauth2.Token{}
	err = json.NewDecoder(tokenFile).Decode(tokenJSON)
	if err != nil {
		log.Printf("Unable to parse token file: %v\n", err)
	}

	credentialsClient := credentialsConfig.Client(context.Background(), tokenJSON)

	return credentialsClient, nil
}
