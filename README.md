# Setting up the bash-cni plugin in a Kubernetes cluster

This is the bash-cni plugin with its DaemonSet to setup the bash-cni plugin in a kubernetes cluster. It uses the 
`bridge` plugin to create veth pairs between the bridge and the containers. The `bridge` plugin calls the `host-local` plugin for 
IP Address Management (IPAM). There is no dependency on nmap or jq. The log file at `/var/log/bash-cni-plugin.log` gives timestamps
in microseconds. The `host-local` plugin also runs much faster than the `nmap` utility. The steps are listed below.

## Create a cluster

We first need a K8s cluster, which can be done by any of the following approaches:
- [Kubeadm](docs/kubeadm.md)
- [Kind](docs/kind.md)

## Install the bash-cni plugin

You can pull the `cericwu/bashcni` image first. Or you could build your own image using the `docker build -t <your_image_name> .` command. Here we use the `cericwu/bashcni` image. Usually we don't need to pull the image. It is used here to make sure that you have access to the docker repository. If not, you need to install docker using the command `sudo apt install docker-ce docker-ce-cli docker.io`.

```
$ docker login
...
Login Succeeded
$ docker pull cericwu/bashcni
...
docker.io/cericwu/bashcni:latest
```

Run the `kubectl apply -f bashcni-ds.yml` command to install the bash-cni plugin.

```
$ kubectl apply -f bashcni-ds.yml
clusterrole.rbac.authorization.k8s.io/bashcni created
clusterrolebinding.rbac.authorization.k8s.io/bashcni created
serviceaccount/bashcni created
daemonset.apps/bashcni created
$ ls -l /etc/cni/net.d
-rw-r--r-- 1 root root 138 May  6 11:42 10-bash-cni-plugin.conf
$ ls -l /opt/cni/bin/bash-cni
-rwxr-xr-x 1 root root 3194 May  6 11:42 /opt/cni/bin/bash-cni
$ ls -l /opt/cni/bin/host-local
-rwxr-xr-x 1 root root 3614480 May  6 11:42 /opt/cni/bin/host-local
```

The output shows the bashcni daemonset is created with its service account and cluster role.
The bash-cni configuration file `10-bash-cni-plugin.conf` is automatically created
and installed in the `/etc/cni/net.d` directory on each node.
The bash-cni script is also installed in the `/opt/cni/bin` directory on each node. It uses
the `host-local` plugin for IP Address Management (IPAM) instead of the `nmap` utility for
generating IP addresses.
The daemonset pods are created in the namespace `kube-system`.
We can run the following command to check on them.


```
$ kubectl get pods -n kube-system
```

The output will show several bashcni pods running, one on each node.

## Deploy a few pods in the kubernetes cluster

We want to deploy a few pods to test the bash-cni plugin.


```
$ kubectl apply -f deploy_monty.yml
pod/monty-vm2 created
pod/monty-vm3 created
$ kubectl run -it --image=busybox pod1 -- sh
$ kubectl get pods -o wide
NAME        READY  STATUS    RESTARTS  AGE  IP           NODE
monty-vm2   1/1    Running   0         5m   10.244.1.4   vm2
monty-vm3   1/1    Running   0         5m   10.244.2.3   vm3
pod1        1/1    Running   0         3m   10.244.2.4   vm3
```

It shows that the pods are installed successfully with the bash-cni plugin. Their IP addresses are also listed in the output.

## Test the connectivities of the pods

We can test the connectivities of the pods as shown below.

```
$ kubectl exec -it pod1 -- sh
/ # ping 10.244.2.3                     // Pod on the same VM
/ # ping 10.244.1.4                     // Pod on a different VM
/ # ping 192.168.122.135                // IP address of the master node
/ # ping 192.168.122.160                // IP address of the host VM
/ # ping 192.168.122.158                // IP address of the other VM
/ # ping 185.125.190.29                 // IP address of ubuntu.com
```

The connection to all the above should be OK after the bashcni daemonset is installed.

## Deleting the daemonset

Deleting the daemonset won't affect the function of the bash-cni plugin.

```
$ kubectl delete -f bashcni-ds.yml
clusterrole.rbac.authorization.k8s.io "bashcni" deleted
clusterrolebinding.rbac.authorization.k8s.io "bashcni" deleted
serviceaccount "bashcni" deleted
daemonset.apps "bashcni" deleted
$ ls -l /etc/cni/net.d
-rw-r--r-- 1 root root 138 May  6 11:42 10-bash-cni-plugin.conf
$ ls -l /opt/cni/bin/bash-cni
-rwxr-xr-x 1 root root 3194 May  6 11:42 /opt/cni/bin/bash-cni
$ ls -l /opt/cni/bin/host-local
-rwxr-xr-x 1 root root 3614480 May  6 11:42 /opt/cni/bin/host-local
```

This is because the bridge, iptables, and ip route entries remain unchanged
after the bashcni daemonset is deleted.
