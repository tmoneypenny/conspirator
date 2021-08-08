### Certificates

#### A note about TLS certs

If the configuration specifies multiple domains, a WC SAN cert will need 
to be issued for each 2nd or 3rd level subdomain.

```
Name=company.com
Name=*.company.com
Name=*.prod.company.com.
Name=*.test.company.com.
```

## DNS validation
`sudo certbot certonly --manual --preferred-challenges dns`

Add the following two records only:
```
_acme-challenge.src.properties TXT Simple "<rec>"
_acme-challenge.default.src.properties TXT simple "<rec>"

challenge 3:
_acme-challenge.default.src.properties TXT simple "<rec>"
                                                  "<rec>"
```

Once certs are installed, configure NS IP to be set to the conspirator server public IP:
```
default.src.properties NS Simple ns1.default.src.properties
ns1.default.src.properties A <ip_of_conspirator>
```