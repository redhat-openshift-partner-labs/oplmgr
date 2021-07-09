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
	"log"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing Hive ClusterDeployment",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		clusterid, err := cmd.Flags().GetString("cluster-id")
		if err != nil {
			log.Printf("Unable to get cluster-id: %v\n", err)
		}

		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Printf("Unable to get namespace: %v\n", err)
		}

		client := HiveClientK8sAuthenticate()

		cdt := &hivev1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      clusterid,
			},
		}

		err = client.Delete(context.Background(), cdt)
		if err != nil {
			log.Printf("Unable to delete cluster deployment %v: %v\n", cdt.Name, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
