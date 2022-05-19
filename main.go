package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func runcmd(path string, args []string, debug bool) (output string, err error) {
    cmd := exec.Command(path, args...)

    var b []byte
    b, err = cmd.CombinedOutput()
    out := string(b)

    if debug {
        fmt.Println(strings.Join(cmd.Args[:], " "))
        if err != nil {
            fmt.Println("runcmd error")
            fmt.Println(out)
        }
    }
    return out, err
}

func splitArray(in []string, num int) [][]string {
    var divided [][]string

    chunk := (len(in) + num - 1) / num
    for i := 0; i < len(in); i += chunk {
        end := i + chunk
        if end > len(in) {
            end = len(in)
        }
        divided = append(divided, in[i:end])
    }
    return divided
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func bash_cni_conf(network string, subnet string, file string) int {
    f, err := os.Create(file) 
    check(err)

    str := fmt.Sprintf("{\n    \"cniVersion\": \"0.3.1\",\n    \"name\": \"mynet\",\n    \"type\": \"bash-cni\",\n    \"network\": \"%s\",\n    \"subnet\": \"%s\"\n}\n", network, subnet)
    num, err := f.WriteString(str)
    check(err)
    f.Sync()
    f.Close()
    return num
}

func bash_cni_paras(network string, subnet []string, addr [][]string, gwcidr string, ipaddr string, idx int, file string) int {
    f, err := os.Create(file) 
    check(err)

    str := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%d\n", network, subnet, addr, gwcidr, ipaddr, idx)
    num, err := f.WriteString(str)
    check(err)
    f.Sync()
    f.Close()
    return num
}

func main() {
    var output, fileName, network, cmdStr, ifname, gw_address, gw_cidr string
    var err error
    var numBytes int
    var args, out, subnet []string
    var address, ifip [][]string
    var match bool = false
    var done int = 0
    var index int = -1

loop:
    args = []string{"-c", "kubectl get nodes -o json | jq -r '.items[].spec.podCIDR'"}
    output, err = runcmd("sh", args, true)
    if err != nil {
        fmt.Println("Error:", output)
    }
    // subnet array 
    subnet = strings.Split(strings.TrimSuffix(output, "\n"), "\n")
    fmt.Println("subnet[]:", subnet)

    args = []string{"-c", "kubectl get nodes -o json | jq -r '.items[].status.addresses[].address'"}
    output, err = runcmd("sh", args, true)
    if err != nil {
        fmt.Println("Error:", output)
    }
    out = strings.Split(strings.TrimSuffix(output, "\n"), "\n")
    address = splitArray(out, len(subnet))
    fmt.Println("address[][]", address)

    if done == 0 {
        // This should always succeed.
        args = []string{"-c", "kubectl get pods -n kube-system -o json | grep cluster-cidr | awk 'BEGIN {FS=\"[ =\\\"]+\"} {print $3}'"}
        output, err = runcmd("sh", args, true)
        if err != nil {
            fmt.Println("Error:", output)
        }
        network = strings.TrimSuffix(output, "\n")
        fmt.Println(network)
        // Find VM name 
        output, err = os.Hostname()
        if err != nil {
            fmt.Println("Error: os.Hostname()")
        }
        fmt.Println("VM name:", output)
        for i := 0; i < len(address); i++ {
            if output == address[i][1] {
                index = i
                break
            }
        }
        fmt.Println("Index:", index)

        // Write to /etc/cni/net.d/10-bash-cni-plugin.conf
        fileName = "/etc/cni/net.d/10-bash-cni-plugin.conf"
        numBytes = bash_cni_conf(network, subnet[index], fileName)
        fmt.Println(numBytes, "bytes written to", fileName)

        // Find the interface name for IP address address[index][0]
        args = []string{"-c", "ip -4 -o a | cut -d ' ' -f 2,7 | cut -d '/' -f 1"}
        output, err = runcmd("sh", args, true)
        if err != nil {
            fmt.Println("Error:", output)
        }
        out = strings.Split(strings.TrimSuffix(output, "\n"), "\n")
        for i := 0; i < len(out); i++ {
            if out[i] != "" {
                rowifip := strings.Split(out[i], " ")
                ifip = append(ifip, rowifip)
            }
        }
        fmt.Println("ifip array:", ifip)

        // Find the ifname
        for i := 0; i < len(ifip); i++ {
            if address[index][0] == ifip[i][1] {
                ifname = ifip[i][0]
                break
            }
        }
        fmt.Println("ifname:", ifname)

        out = strings.Split(subnet[index], "/")
        ip_address := out[0]
        mask_size := out[1]
        fmt.Println("mask_size:",mask_size)

        slen := len(ip_address)
        if slen > 0 && ip_address[slen - 1] == '0' {
            gw_address = ip_address[:slen - 1] + "1"
        }
        gw_cidr = gw_address + "/" + mask_size
        fmt.Println(ip_address, gw_address, gw_cidr)

        args = []string{"link", "show", "cni0"}
        _, err = runcmd("ip", args, true)
        if err == nil {
            args = []string{"addr", "add", gw_cidr, "dev", "cni0"}
            _, err = runcmd("ip", args, true)
            if err != nil {
                fmt.Println("Error: add cni0 address" )
            }
        } else {
            // Add bridge cni0
            args = []string{"link", "add", "cni0", "type", "bridge"}
            _, err = runcmd("ip", args, true)
            if err != nil {
                fmt.Println("Error: create cni0" )
            }
            args = []string{"link", "set", "cni0", "up"}
            _, err = runcmd("ip", args, true)
            if err != nil {
                fmt.Println("Error: bringup cni0" )
            }
            args = []string{"addr", "add", gw_cidr, "dev", "cni0"}
            _, err = runcmd("ip", args, true)
            if err != nil {
                fmt.Println("Error: add cni0 address" )
            }
        }

        // Add iptables entries
        cmdStr = fmt.Sprintf("iptables -t filter -A FORWARD -s %s -j ACCEPT", network)
        args = []string{"-c", cmdStr}
        _, err = runcmd("sh", args, true)
        if err != nil {
            fmt.Println("Error:", cmdStr)
        }
        cmdStr = fmt.Sprintf("iptables -t filter -A FORWARD -d %s -j ACCEPT", network)
        args = []string{"-c", cmdStr}
        _, err = runcmd("sh", args, true)
        if err != nil {
            fmt.Println("Error:", cmdStr) 
        }
        cmdStr = fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s ! -o cni0 -j MASQUERADE", subnet[index])
        args = []string{"-c", cmdStr}
        _, err = runcmd("sh", args, true)
        if err != nil {
            fmt.Println("Error:", cmdStr)
        }
        done = 1
    }

    // Setup ip route to other VMs
    args = []string{"route"}
    output, err = runcmd("ip", args, true)
    if err != nil {
        fmt.Println("Error:", cmdStr)
    }
    out = strings.Split(strings.TrimSuffix(output, "\n"), "\n")
    for i := 0; i < len(subnet); i++ {
        if i != index && subnet[i] != "" {
            cmdStr = fmt.Sprintf("%s via %s dev %s", subnet[i], address[i][0], ifname)
            match = false
            for j := 0; j < len(out); j++ {
                if strings.Contains(out[j], cmdStr) {
                    match = true
                    break
                }
            }
            if match == false {
                args = []string{"route", "add", subnet[i], "via", address[i][0], "dev", ifname}
                _, err = runcmd("ip", args, true)
                if err != nil {
                    fmt.Println("Error: ip route add", subnet[i], "via", address[i][0], "dev", ifname)
                }
            }
        }
    }
    
    args = []string{"-c", "kubectl get nodes -w"}
    output, err = runcmd("sh", args, true)
    if err != nil {
        fmt.Println("Error:", output)
    }
    goto loop
}
