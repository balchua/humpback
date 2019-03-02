# Pod-Runner

## Overview
A simple pod scheduler which waits for the pod's result.  
This tool can help external Job scheduler to trigger pod execution.

Cases wherein you need to execute commands inside an image, this tool can come in handy to make sure you are able to execute a completable command inside your image without the need to stand up an SSH daemon for Job orchestrator to connect to.

This will also allow a more granular resource allocation per job.  This results to better kubernetes cluster resource utilization.

## Configuration

There is a minimum configuration this tool requires.

1. pod-runner.yaml

This configuration should contain all the "applications" that the pod runner can schedule.

Sample configuration:

    applications:
    - name: app1
      template: /app1-templates/pod1.tmpl
      container:
        resource-requests:
          memory: 10Mi
          cpu: 
        resource-limits:
          memory: 50Mi
          cpu:
        image: balchua/app1:1.0
        uid: 1000
        gid: 1000
    - name: app2
      template: /app2-templates/pod1.tmpl
      container:
        resource-requests:
          memory: 10Mi
          cpu: 200m
        resource-limits:
          memory: 50Mi
          cpu: 100m
        image: balchua/demo:1.0
        uid: 2000
        gid: 2000

2. Pod templating

There is a need to provide a pod template, which will allow the runner to dynamically interpolate pod commands, memory and cpu (requests / limits) and image to use.

Sample template:

```
apiVersion: v1
kind: Pod
metadata:
  name: {{ .Name }}-{{ .UniqueId }}
  labels:
    app: {{ .Name }}
    appUnique: {{ .Name }}-{{ .UniqueId }}
spec:
  containers:
  - image: {{ .Container.Image }}
    args: [{{ .Container.Arguments }}]
    imagePullPolicy: IfNotPresent
    name: {{ .Name }}
    securityContext:
      runAsUser: {{ .Container.UID }}
      fsGroup: {{ .Container.GID }}
```

*Some few notes to remember the pod-runner will automatically set the `restartPolicy` to `Never` otherwise the Pod will go into `CrashLoop`  If possible, set the `activeDeadlineSeconds` to make sure that the Pod does not run indefinitely.*

## Executing the pod-runner



```
pod-runner --application [your application name] --namespace [the namespace where you want the job to run] --command [the Job's command] --kube-config $KUBECONFIG

```

Where:

* `--application` should be the name of the application defined in the `pod-runner.yaml`

* `--namespace` which namespace the pod will be scheduled.

* `--command` The actual command to run.

* `--kube-config`  If you are running this outside the Kubernetes cluster, specify this, otherwise leave it empty.

*Any Pod Status, that results to other than `Successful` will be returned as an error i.e. `exit(1)`*

## Cleanup

The controller will automatically delete the pod, as soon as it completes or failed.  The pod logs are tailed to the console.

## Limitations

The command to run will not be able to interpolate any environment variables defined at the Container `args`.

For example:

`/usr/src/app/main-good.sh $APP_HOME` - the `$APP_HOME` will not be interpolated.  In order to mitigate this,  the `$APP_HOME` must be resolved from inside the shell script, in this case `/usr/src/app/main-good.sh`

~~If the Pod goes into ImagePullError, the pod runner does not appear to detect this, leaving it in Pending mode.  It will not complete.~~
This is now supported.  While the PodStatus is in Pending state, and if the containerStatus is not `ContainerCreating` its assumed to be an `ImagePullError`.


## Build

* Clone the repo.  

* Enable GO module  `export GO111MODULE=on`

* Run `go build`

## Examples

Check out the directory `examples`


    


