### oplmgr

CLI application to work with OpenShift Partner Lab components.  

```
Usage:
  oplmgr SUB-COMMAND [flags]
  oplmgr [command]

Available Commands:
  delete      Delete an existing Hive ClusterDeployment
  email       Send email to contacts of cluster
  help        Help about any command
  info        Get information about cluster(s)
  provision   Create a Hive ClusterDeployment
  sleep       Set powerState of Hive ClusterDeployment to Hibernating
  version     Version of oplmgr
  wake        Set powerState of Hive ClusterDeployment to Running

Flags:
      --cluster-id string   id of cluster to interact with
      --company string      company name provided by request form (default "opl")
  -h, --help                help for oplmgr
      --namespace string    namespace to interact with (default "hive")

Use "oplmgr [command] --help" for more information about a command.
```
