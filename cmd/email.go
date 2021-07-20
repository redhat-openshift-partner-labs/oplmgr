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
	"context"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"os"
	"strings"
)

var welcome bool
var credentials bool
var kubeadmin bool
var kubeconfig bool

var to []string
var cc []string
var bcc []string

func init() {
	flags := emailCmd.Flags()
	flags.BoolVar(&welcome, "welcome", false, "send welcome email")
	flags.BoolVar(&credentials, "credentials", false, "send only credentials")
	flags.BoolVar(&kubeadmin, "kubeadmin", false, "send only kubeadmin password")
	flags.BoolVar(&kubeconfig, "kubeconfig", false, "send only kubeconfig")
	flags.StringSliceVar(&to, "to", []string{}, "comma separated list of to addresses")
	flags.StringSliceVar(&cc, "cc", []string{}, "comma separated list of cc addresses")
	flags.StringSliceVar(&bcc, "bcc", []string{}, "comma separated list of bcc addresses")

	flags.String("company", "", "provide company name for email subject line")
	flags.String("clusterid", "", "provide clusterid you want to work with")

	emailCmd.MarkFlagRequired("company")
	emailCmd.MarkFlagRequired("clusterid")

	rootCmd.AddCommand(emailCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// emailCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// emailCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getClusterDeploymentInfo(clusterid string) (consoleurl string, timezone string, kubeadminlink *v1.Secret, kubeconfiglink *v1.Secret){
	cd := hivev1.ClusterDeployment{}

	hiveclient := HiveClientK8sAuthenticate()
	err := hiveclient.Get(context.Background(), types.NamespacedName{Namespace: "hive", Name: clusterid}, &cd); if err != nil {
		log.Printf("Unable to get the cluster with id %v: %v\n", clusterid, err)
	}

	k8sclient := K8sAuthenticate()
	kubeadminsecret, err := k8sclient.CoreV1().Secrets("hive").Get(context.Background(), cd.Spec.ClusterMetadata.AdminPasswordSecretRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Unable to get the cluster kubeadmin secret: %v\n", err)
	}

	kubeconfigsecret, err := k8sclient.CoreV1().Secrets("hive").Get(context.Background(), cd.Spec.ClusterMetadata.AdminKubeconfigSecretRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Unable to get the cluster kubeconfig secret: %v\n", err)
	}

	return cd.Status.WebConsoleURL, cd.ObjectMeta.Labels["timezone"], kubeadminsecret, kubeconfigsecret
}


// emailCmd represents the email command
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Send email to contacts of cluster",
	Long: `oplmgr email --welcome --clusterid b592ec70-487f-44fc-a389-80bbf111ec96 --company "Red Hat" --to a@a.com,b@a.com --cc c@a.com,d@a.com --bcc e@a.com
oplmgr email --credentials --clusterid b592ec70-487f-44fc-a389-80bbf111ec96 --company "Red Hat" --to a@a.com,b@a.com --cc c@a.com,d@a.com --bcc e@a.com
oplmgr email --kubeadmin --clusterid b592ec70-487f-44fc-a389-80bbf111ec96 --company "Red Hat" --to a@a.com,b@a.com --cc c@a.com,d@a.com --bcc e@a.com
oplmgr email --kubeconfig --clusterid b592ec70-487f-44fc-a389-80bbf111ec96 --company "Red Hat" --to a@a.com,b@a.com --cc c@a.com,d@a.com --bcc e@a.com

This command will silently fail on multiple occassions. You must provide a type as listed below and in the examples,
clusterid, company, and one of --to --cc or --bcc for this command to work properly. You will not get an error in most
cases so provide all appropriate and required flags to succeed.

Send various types of email to contacts listed on the cluster. You will not be
able to set or see the email addresses through this command as it only takes the cluster-id
and the email type you want to send:

welcome - provides the initial welcome email after cluster has been successfully provisioned.
credentials - sends the kubeadmin password and kubeconfig to contacts via privatebin links.
kubeadmin - send only the kubeadmin password via privatebin link.
kubeconfig - send only the kubeconfig via privatebin link.

If you pass more than one type of email flag the first will get created and sent only in the order
listed above; welcome, credentials, kubeadmin, kubeconfig.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterid, err := cmd.Flags().GetString("clusterid")
		octet := strings.Split(clusterid, "-")[0]
		company, err := cmd.Flags().GetString("company")
		consoleurl, timezone, kubeadminsecret, kubeconfigsecret := getClusterDeploymentInfo(clusterid)
		clusterinfo := GenerateMultiplePastes(os.Getenv("PRIVATEBIN_HOST"),
			map[string]string{
			"kubeadmin": string(kubeadminsecret.Data["password"]),
			"kubeconfig": string(kubeconfigsecret.Data["raw-kubeconfig"]),
			})
		clusterinfo["consoleurl"] = consoleurl
		clusterinfo["clusterid"] = octet
		clusterinfo["company"] = company
		clusterinfo["timezone"] = timezone

		sendwelcome, err := cmd.Flags().GetBool("welcome")
		if err != nil {
			log.Printf("Error trying to get welcome flag: %v\n", err)
		}

		sendcreds, err := cmd.Flags().GetBool("credentials")
		if err != nil {
			log.Printf("Error trying to get credentials flag: %v\n", err)
		}

		sendadmin, err := cmd.Flags().GetBool("kubeadmin")
		if err != nil {
			log.Printf("error trying to get kubeadmin flag: %v\n", err)
		}

		sendconfig, err := cmd.Flags().GetBool("kubeconfig")
		if err != nil {
			log.Printf("Error trying to get kubeconfig flag: %v\n", err)
		}

		switch {
		case sendwelcome:
			SendWelcomeEmail(&to, &cc, &bcc, clusterinfo)
			break
		case sendcreds:
			SendCredsEmail(&to, &cc, &bcc, clusterinfo)
			break
		case sendadmin:
			SendAdminEmail(&to, &cc, &bcc, clusterinfo)
			break
		case sendconfig:
			SendConfigEmail(&to, &cc, &bcc, clusterinfo)
			break
		}
	},
}

