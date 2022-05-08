# Setting up the bash-cni plugin in a Kubernetes cluster

This is the bash-cni plugin with its DaemonSet to setup the bash-cni plugin in a kubernetes cluster.
The steps are listed below.

## Create a kubernetes cluster

Docker comes with the native cgroup driver cgroupfs on Ubuntu. Modify the `10-kubeadm.conf` file in the
`/etc/systemd/system/kubelet.service.d` directory on all VMs.  The `kubeadm init ...` command may fail on
the master node if this is not corrected.

```
$ sudo vi /etc/systemd/system/kubelet.service.d/10-kubeadm.conf
// Add " --cgroup-driver=cgroupfs" to the end of the environment variable
// Environment="KUBELET_KUBECONFIG_ARGS..."
```

Manually start the cluster on the master node.

```
$ sudo kubeadm init --pod-network-cidr=10.244.0.0/16
...                     // The output includes the "kubeadm join ..." command for worker nodes to join
kubeadm join 192.168.122.135:6443 –-token rbmp1g.pg798r0cshk4qvbw \
        –-discovery-token-ca-cert-hash sha256:5888010a3c67f2842279e5a22c496e3945e6ff6678f0ccae8d4cf03a350de4c01
```

## Run the `kubeadm join ...` command on worker nodes

Run the `kubeadm join ...` command on each node to join the cluster.

```
$ sudo kubeadm join 192.168.122.135:6443 –-token rbmp1g.pg798r0cshk4qvbw \
        –-discovery-token-ca-cert-hash sha256:5888010a3c67f2842279e5a22c496e3945e6ff6678f0ccae8d4cf03a350de4c01
...
This node has joined the cluster...
Run 'kubectl get nodes' on the control-plane to see this node join the cluster.
```

## Check the cluster on the master node.

Copy the configuration file to your `~/.kube` directory and run the `kubectl get nodes` command.

```
$ mkdir -p $HOME/.kube
$ sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
$ sudo chown $(id -u):$(id -g) $HOME/.kube/config
$ kubectl get nodes
```

The status of each node will be shown as `NotReady` at this point. It is because there is no CNI plugin
installed yet.

## Install the bash-cni plugin

You can pull the `cericwu/bashcni` image first. Or you could build your own image using the `docker build -t <your_image_name> .` command. Here we use the `cericwu/bashcni` image. Usually we don't need to pull the image. It is used here to make sure that you have access to the docker repository.

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
-rwxr-xr-x 1 root root 3389 May  6 11:42 /opt/cni/bin/bash-cni
```

The output shows the bashcni daemonset is created with its service account and cluster role.
The bash-cni configuration file `10-bash-cni-plugin.conf` is automatically created
and installed in the `/etc/cni/net.d` directory on each node.
The bash-cni script is also installed in the `/opt/cni/bin` directory on each node.
The daemonset pods are created in the namespace kube-system.
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

It shows the pods and their IP addresses.

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
-rwxr-xr-x 1 root root 3389 May  6 11:42 /opt/cni/bin/bash-cni
```

This is because the bridge, iptables, and ip route entries remain unchanged
after the bashcni daemonset is deleted.
