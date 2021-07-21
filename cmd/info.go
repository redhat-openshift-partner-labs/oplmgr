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
	"fmt"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about cluster(s)",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		clusterid, err := cmd.Flags().GetString("clusterid")
		if err != nil {
			log.Printf("Unable to get the clusterid flag: %v\n", err)
		}

		cds := make(map[string]interface{})
		cd := make(map[string]interface{})

		if clusterid == "" {
			cds = getClusterDeployments("")
			for name, info := range cds {
				if info.(map[string]interface{})["cluster"].(hivev1.ClusterDeployment).Spec.PowerState != "Hibernating" {
					info.(map[string]interface{})["powerstate"] = "Running"
				}

				_, err = fmt.Fprintf(os.Stdout, `
------------------------------------------------
Cluster ID: %s
Request URL: %s
Cluster State: %s
Console URL: %s
Credentials:
  kubeadmin: %s
  kubeconfig: %s
------------------------------------------------
`, info.(map[string]interface{})["cluster"].(hivev1.ClusterDeployment).Name, "https://ui.apps.eng.partner-lab.rhecoeng.com/request/"+name,
					info.(map[string]interface{})["powerstate"],
					info.(map[string]interface{})["consoleurl"],
					info.(map[string]interface{})["credentials"].(map[string]string)["kubeadmin"],
					info.(map[string]interface{})["credentials"].(map[string]string)["kubeconfig"])

				if err != nil {
					log.Printf("Unable to print information on cluster %v: %v\n", name, err)
				}
			}
		} else {
			cd = getClusterDeployments(clusterid)

			if cd["cluster"].(hivev1.ClusterDeployment).Spec.PowerState != "Hibernating" {
				cd["powerstate"] = "Running"
			}

			_, err = fmt.Fprintf(os.Stdout, `
------------------------------------------------
Cluster ID: %s
Request URL: %s
Cluster State: %s
Console URL: %s
Credentials:
  kubeadmin: %s
  kubeconfig: %s
------------------------------------------------
`, cd["cluster"].(hivev1.ClusterDeployment).Name, "https://ui.apps.eng.partner-lab.rhecoeng.com/request/"+clusterid,
				cd["powerstate"],
				cd["consoleurl"],
				cd["credentials"].(map[string]string)["kubeadmin"],
				cd["credentials"].(map[string]string)["kubeconfig"])

			if err != nil {
				log.Printf("Unable to print information on cluster %v: %v\n", clusterid, err)
			}
		}
	},
}

func getClusterDeployments(clusterid string) map[string]interface{} {
	hiveclient := HiveClientK8sAuthenticate()

	if clusterid != "" {
		cd := hivev1.ClusterDeployment{}

		err := hiveclient.Get(context.Background(), types.NamespacedName{Namespace: "hive", Name: clusterid}, &cd)
		if err != nil {
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

		clusterinfo := make(map[string]interface{})

		clusterinfo["credentials"] = GenerateMultiplePastes(os.Getenv("PRIVATEBIN_HOST"),
			map[string]string{
				"kubeadmin":  string(kubeadminsecret.Data["password"]),
				"kubeconfig": string(kubeconfigsecret.Data["raw-kubeconfig"]),
			})

		clusterinfo["cluster"] = cd
		clusterinfo["consoleurl"] = cd.Status.WebConsoleURL
		clusterinfo["timezone"] = cd.ObjectMeta.Labels["timezone"]

		return clusterinfo

	} else {
		cds := hivev1.ClusterDeploymentList{}
		opts := client.ListOptions{Namespace: "hive"}

		err := hiveclient.List(context.Background(), &cds, &opts)
		if err != nil {
			log.Printf("Unable to get the cluster deployments from namespace hive: %v\n", err)
		}

		k8sclient := K8sAuthenticate()
		clusters := make(map[string]interface{})

		for _, cluster := range cds.Items {
			clusterinfo := make(map[string]interface{})

			kubeadminsecret, err := k8sclient.CoreV1().Secrets("hive").Get(context.Background(), cluster.Spec.ClusterMetadata.AdminPasswordSecretRef.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("Unable to get the cluster kubeadmin secret: %v\n", err)
			}

			kubeconfigsecret, err := k8sclient.CoreV1().Secrets("hive").Get(context.Background(), cluster.Spec.ClusterMetadata.AdminKubeconfigSecretRef.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("Unable to get the cluster kubeconfig secret: %v\n", err)
			}

			clusterinfo["credentials"] = GenerateMultiplePastes(os.Getenv("PRIVATEBIN_HOST"),
				map[string]string{
					"kubeadmin":  string(kubeadminsecret.Data["password"]),
					"kubeconfig": string(kubeconfigsecret.Data["raw-kubeconfig"]),
				})

			clusterinfo["cluster"] = cluster
			clusterinfo["consoleurl"] = cluster.Status.WebConsoleURL
			clusterinfo["timezone"] = cluster.ObjectMeta.Labels["timezone"]

			clusters[cluster.ObjectMeta.Name] = clusterinfo
		}

		return clusters
	}
}

func init() {
	flags := infoCmd.Flags()
	flags.String("clusterid", "", "return information about a cluster")

	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
