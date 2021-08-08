### DNS - address already in use

If you get an error like `listen udp :53: bind: address already in use`,
trying to start the server on Ubuntu 18.04+, you'll most likely need to 
free up the port used by systemd-resolved.  

Edit `/etc/systemd/resolved.conf`
```
...
# Uncomment and change the following to match:
DNSStubListener=no
...
```
`ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf && shutdown -r now`

#### PublicAddress

PublicAddress is used to set the public facing address when the listener
IP for any service is set to `""`. If you do not set a listener for any
service to be `""`, then you can omit setting this field in the config.