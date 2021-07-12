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

	. "github.com/redhat-openshift-partner-labs/oplmgr/internal"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var sleepCmd = &cobra.Command{
	Use:   "sleep",
	Short: "Set powerState of Hive ClusterDeployment to Hibernating",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		clusterid, err := cmd.Flags().GetString("clusterid")
		if err != nil {
			log.Printf("Unable to get clusterid: %v\n", err)
		}

		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Printf("Unable to get clusterid: %v\n", err)
		}

		client := HiveClientK8sAuthenticate()

		cdo := &hivev1.ClusterDeployment{}

		cdt := types.NamespacedName{
			Namespace: namespace,
			Name:      clusterid,
		}

		if err = client.Get(context.TODO(), cdt, cdo); err != nil {
			log.Printf("Unable to get cluster deployment: %v\n", err)
		}

		cdo.Spec.PowerState = "Hibernating"

		if err = client.Update(context.Background(), cdo); err != nil {
			log.Printf("Unable to update cluster deployment powerState: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(sleepCmd)
}
