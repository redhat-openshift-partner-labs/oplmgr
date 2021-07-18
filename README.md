### oplmgr

CLI application to work with OpenShift Partner Lab components.  

```
Usage:
  oplmgr SUB-COMMAND [flags]
  oplmgr [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  delete      Delete an existing Hive ClusterDeployment
  email       Send email to contacts of cluster
  help        Help about any command
  info        Get information about cluster(s)
  provision   Create a Hive ClusterDeployment
  report      Generate various reports for OpenShift Partner Labs
  sleep       Set powerState of Hive ClusterDeployment to Hibernating
  version     Version of oplmgr
  wake        Set powerState of Hive ClusterDeployment to Running

Flags:
      --clusterid string   id of cluster to interact with
      --company string     company name provided by request form (default "redhat")
  -h, --help               help for oplmgr
      --namespace string   namespace to interact with (default "hive")

Use "oplmgr [command] --help" for more information about a command.
```
