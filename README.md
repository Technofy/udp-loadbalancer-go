# UDP IPSec Load balancer in go

Simple UDP Load-Balancer which also supports IPSec.

The load balancing is done using IP Source hash.

## Features
 - UDP load balacing with or without IPSEC
 - Loadbalacing to static IPs or AWS Autoscaling Group
 - Software health metric on AWS CloudWatch
 
 
## Configuration

The file `config.yaml` contains your application configuration. The one present by default contains some example to get you started.

- Upstream

> This section contains all the target groups. They can be either IP addresses, or AWS Auto-scaling Group.

- Servers

> This section contains how is the traffic load balanced.

* Pacemaker

> This object tell the Load Balancer where to update it's health status. This is allowing us to monitor it trough CloudWatch.

Here's a example config:

```yaml
###################
# Configuration
###################
## List of servers to load balance to

# Load Balancing to AWS autoscaling group
upstreams:
 - name: vpn_servers
   type: aws_autoscaling_group
   hash: remote_ip
   targets:
    - myAutoScalingGroup

# Load Balancing to static servers
 - name: static_servers
   hash: remote_ip
   targets:
    - 10.0.0.2
    - 10.0.0.3
    - 10.0.0.4

## Load balancer Listeners and redirect
servers:
  - bind: 0.0.0.0
    port: 500
    proto: udp
    pass: static_servers:500

  - bind: 0.0.0.0
    port: 4500
    proto: udp
    pass: static_servers:4500


## Cloudwatch metric to monitor the health of the load balancer
pacemaker:
  region: eu-west-1
  interval: 10
  namespace: IPSecLB
  metric: Healthy
```

### IAM Actions required:

* cloudwatch:putMetricData
* ec2:DescribeInstances
* autoscaling:DescribeAutoScalingGroups

Example role with access on all ressources:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1511423354553",
      "Action": [
        "cloudwatch:putMetricData",
        "ec2:DescribeInstances",
        "autoscaling:DescribeAutoScalingGroups"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}

```


## License

Copyright 2017 Technofy

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.