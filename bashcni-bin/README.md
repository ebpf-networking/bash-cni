# Replace the bash-cni shell script with bash-cni executable

This is to replace the bash-cni shell script with a bash-cni executable. The bash-cni shell script uses nmap and jq,
which need to be installed in all cluster nodes first. The new bash-cni executable was written in go. It does not
need either nmap or jq for execution.

## Use the bridge plugin

The bash-cni executable uses the bridge plugin to create the cni0 bridge, if necessary. It then calls the
host-local plugin for IP Address Management (IPAM). VMs.  The code below shows an example of using the bridge
plugin.

```
$ echo "{\"cniVersion\": \"0.3.1\", \"type\": \"bash-cni\", \"name\": \"$myname\", \"ipam\": { \"type\":  \"host-local\", \"subnet\": \"$subnet\" }}" | CNI_COMMAND=ADD CNI_CONTAINERID=$CNI_CONTAINERID CNI_NETNS=$CNI_NETNS CNI_IFNAME=$CNI_IFNAME CNI_PATH=/opt/cni/bin /opt/cni/bin/bridge
{
    "cniVersion": "0.3.1",
    "interfaces": [
        {
            "name": "cni0",
            "mac": "9e:ea:f9:68:4f:54"
        }
        {
            "name": "vethc27584a9",
            "mac": "9e:ea:f9:68:4f:54"
        }
        {
            "name": "veth0",
            "mac": "ee:19:66:b6:6a:02",
            "sandbox": "xxxxxxx"
        }
    ],
    "ips": [
        {
            "version": "4",
            "interface": 2,
            "address": "10.244.0.3/24",
            "gateway": "10.244.0.1"
        }
    ],
    "dns": {}
}
```

The host-local plugin provides the address `10.244.0.3` for the newly created container. It also shows that
the gateway IP address is `10.244.0.1`.

The advantages of using go and the bridge plugin include the following:
- It calls the host-local plugin for IPAM.
- It eliminates most of the ip commands.
- There is no dependency on nmap or jq.
- The speed is drastically improved.

## The Steps for CNI_COMMAND ADD

The code becomes much simpler using the bridge plugin. The steps for ADD include
- Read the configuration file `/etc/cni/net.d/10-bash-cni-plugin.conf`, which is created by the daemonset for each node. We get the network CIDR, subnet CIDR, and name from the configuration file.
- Call the bridge plugin. It provides the guest interface MAC address, container IP address, and gateway IP address.
- Add an `ip route` entry in the container.
- Output the needed information to stdout.

## The Steps for CNI_COMMAND DEL

- Read the configuration file to get network CIDR, subnet CIDR, and the name. The name is `mynet`.
- Get the container IP address.
- Delete the entry in the `/var/lib/cni/networks/mynet` directory, which is managed by the host-local plugin.
