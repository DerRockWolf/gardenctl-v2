## gardenctl config set-garden

Modify or add a Garden to the gardenctl configuration

### Synopsis

Modify or add a Garden to the gardenctl configuration.
A valid Garden configuration consists of a name (required), kubeconfig path (required), a context as well as any number of patterns.
In order to share the configuration with gardenlogin, you need to set the name to the cluster identity.

```
gardenctl config set-garden [flags]
```

### Examples

```
# add new Garden my-garden with no additional values
gardenctl config set-garden my-garden

# add new Garden with name set to cluster identity and path to kubeconfig file configured
export KUBECONFIG=~/path/to/garden-cluster/kubeconfig.yaml
CLUSTER_IDENTITY=$(kubectl -n kube-system get configmap cluster-identity -ojsonpath={.data.cluster-identity})
gardenctl config set-garden $CLUSTER_IDENTITY --kubeconfig $KUBECONFIG

# configure my-garden with a context and patterns
gardenctl config set-garden my-garden --context garden-context --pattern "^(?:landscape-dev/)?shoot--(?P<project>.+)--(?P<shoot>.+)$" --pattern "https://dashboard\.gardener\.cloud/namespace/(?P<namespace>[^/]+)/shoots/(?P<shoot>[^/]+)
```

### Options

```
      --alias string          unique alias of this Garden that can be used instead of the name to target this Garden
      --context string        override the current-context of the garden cluster kubeconfig
  -h, --help                  help for set-garden
      --kubeconfig string     path to kubeconfig file for this Garden cluster
      --pattern stringArray   define regex match patterns for this garden for custom input formats for targeting.
                              Use named capturing groups to match target values.
                              Supported capturing groups: project, namespace, shoot.
                              Note that if you set this flag it will overwrite the pattern list in the config file.
                              You may specify any number of extra patterns.
```

### Options inherited from parent commands

```
      --add-dir-header                   If true, adds the file directory to the header of the log messages
      --alsologtostderr                  log to standard error as well as files (no effect when -logtostderr=true)
      --config string                    config file (default is ~/.garden/gardenctl-v2.yaml)
      --log-backtrace-at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log-dir string                   If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log-file string                  If non-empty, use this log file (no effect when -logtostderr=true)
      --log-file-max-size uint           Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --one-output                       If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --skip-headers                     If true, avoid header prefixes in the log messages
      --skip-log-headers                 If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity         logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=false) (default 2)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [gardenctl config](gardenctl_config.md)	 - Modify gardenctl configuration file using subcommands

