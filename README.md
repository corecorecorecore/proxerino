# proxerino
Golang Proxy Checker via file, zmap or masscan


```bash
zmap -p port -q | ./scanner outputfile port <threads>
masscan -p port1,port2 0.0.0.0/0 --rate=99999999 --exclude 255.255.255.255 | awk '{print $6":"$4}' | sed 's/\/tcp//g' | ./scanner outputfile <threads>
cat ips.txt | ./scanner outputfile port <threads>
cat proxies.txt | ./scanner outputfile <threads>
```
