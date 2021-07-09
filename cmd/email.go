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
	"fmt"

	"github.com/spf13/cobra"
)

var welcome bool
var credentials bool
var kubeadmin bool
var kubeconfig bool

// emailCmd represents the email command
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Send email to contacts of cluster",
	Long: `Send various types of email to contacts listed on the cluster. You will not be
able to set or see the email addresses through this command as it only takes the cluster-id
and the email type you want to send:

welcome - provides the initial welcome email after cluster has been successfully provisioned.
credentials - sends the kubeadmin password and kubeconfig to contacts via privatebin links.
kubeadmin - send only the kubeadmin password via privatebin link.
kubeconfig - send only the kubeconfig via privatebin link.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("email called")
	},
}

func init() {
	flags := emailCmd.Flags()
	flags.BoolVar(&welcome, "welcome", false, "send welcome email")
	flags.BoolVar(&credentials, "credentials", false, "send only credentials")
	flags.BoolVar(&kubeadmin, "kubeadmin", false, "send only kubeadmin password")
	flags.BoolVar(&kubeconfig, "kubeconfig", false, "send only kubeconfig")

	rootCmd.AddCommand(emailCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// emailCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// emailCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
