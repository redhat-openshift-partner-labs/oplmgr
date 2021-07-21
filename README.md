## oplmgr

CLI application to work with OpenShift Partner Lab components.  
<br />

#### NOTE

When using the email command you will need to provide some environment variables:

You will need access to a privatebin host, we provide one by default, but you still need to set the environment
variable.  
PRIVATEBIN_HOST=https://bin.apps.eng.partner-lab.rhecoeng.com

You will use the same kubeconfig but two different environment variables; make sure both are set.  
OPENSHIFT_KUBECONFIG=$HOME/.kube/config  
KUBECONFIG=$HOME/.kube/config

You can use the same address for SMTP_USER and SMTP_FROM or not; this will be a configuration requirement known by
you.  
SMTP_PASSWORD=mypassword  
SMTP_USER=smtpuser@mydomain.com  
SMTP_FROM=smtpemail@mydomain.com  
SMTP_HOST=smtphost.mydomain.com

```
Program to manipulate provisioning, sleep, wake, and deletion of clusters
created using OpenShift Partner Labs. You will need the cluster_id and timezone
to successfully use this application:

oplmgr provision --clusterid 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr delete --clusterid 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr sleep --clusterid 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr wake --clusterid 177933cc-e47a-42fc-9ad9-f6efbb27f44a

Provisioning happens at 7am of the timezone provided. Here are the timezones we set:

UTC-5    'America/Panama'    americas
UTC+1    'Africa/Algiers'    emea
UTC+7    'Asia/Jakarta'      apac

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
