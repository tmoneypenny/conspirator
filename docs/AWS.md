## AWS

### EC2

#### Instance Settings
When running conspirator on EC2, it's best to allocate an EIP to the instance. This is because conspirator is configured to be an authoritive name server for the FQDN in the server's configuration.

Resource Requirements:
| Resource | Minimum | Recommended |
| -------- | ------- | ----------- |
| Memory | 100Mb | 1Gb |
| CPU | 1vCPU | 1vCPU |
| Disk | 500Mb | 1Gb |

Based on the above requirements, most small-medium use-cases should be able to run on the `t2.micro` instancee type. 


### Route53

Hosted Zone for `example.com` and Conspirator configured to use `test.example.com` as its main zone:

| Name | Type | Value |
| ---- | ---- | ----- |
| example.com | NS | awsdns.com |
| example.com | SOA | ns.awsdns.com |
| test.example.com | NS | ns1.test.example.com |
| ns1.test.example.com | A | &lt;IP of Conspirator&gt; |
| _acme-challenge.example.com | TXT | "challenge" |