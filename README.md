# Talos KMS Server

This project implements a [Talos KMS API](https://github.com/siderolabs/kms-client) that uses the AWS KMS as the actual KMS solution.

Features:
* KMS Server proxy for AWS KMS
* Automated TLS Certificate creation and rotation with Let's Encrypt

## Building
To build the project run the make command:
```bash
$ make build
```

The binary will be built inside of the `bin` directory.

## Operations

### Requirements
To run the KMS Server, you need to provide it with AWS credentials and an existing AWS KMS KeyID.

The AWS user needs to have the following policy attached to it:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "iam:CreateServiceLinkedRole",
                "kms:Decrypt",
                "kms:TagResource",
                "kms:Encrypt",
                "kms:PutKeyPolicy",
                "kms:DescribeKey",
                "kms:CreateKey"
            ],
            "Resource": "*"
        },
        {
            "Action": "route53:GetChange",
            "Effect": "Allow",
            "Resource": "arn:aws:route53:::change/*"
        },
        {
            "Action": "route53:ChangeResourceRecordSets",
            "Effect": "Allow",
            "Resource": [
                "arn:aws:route53:::hostedzone/AWS_DNS_ZONE_ID"
            ]
        },
        {
            "Action": [
                "route53:ListResourceRecordSets",
                "route53:ListHostedZonesByName",
                "route53:ListHostedZones"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

### Running
You can run the server like this:
```bash
$ export AWS_KMS_KEY_ID=$AWS_KMS_KEY_ID
$ export AWS_ACCESS_KEY_ID=$ACCESS_KEY_ID
$ export AWS_SECRET_ACCESS_KEY=$ACCESS_KEY_SECRET
$ export AWS_HOSTED_ZONE_ID=$HOSTED_ZONE_ID

$ taloskms --domain kms.dev.example.com
```
This will start the server and listen for incomming messages on `*:4050`.

### Usage command:
```bash
$ taloskms -h
NAME:
   taloskms - Talos KMS Server

USAGE:
   taloskms [global options]

VERSION:
   0.2.0-SNAPSHOT-3c77f4c

GLOBAL OPTIONS:
   --listen-port value, -p value                          Service listen port (default: ":4050") [$LISTEN_PORT]
   --email value, -e value                                Email to use for ACME Client [$EMAIL]
   --domain value, -d value [ --domain value, -d value ]  Domain used in SAN filed for the server certificate (can be repeated) [$DOMAINS]
   --cron value, -c value                                 CRON string for certificate renewal cronjob (default: "0 0 1 */2 *") [$CRON]
   --workdir value, --wd value                            Working directory to store files (default: ".taloskms") [$WORKDIR]
   --log-level value, -l value                            Logging level to use (default: "info") [$LOG_LEVEL]
   --aws-kms-key-id value                                 AWS KMS key ID [$AWS_KMS_KEY_ID]
   --aws-access-key-id value                              AWS access key ID [$AWS_ACCESS_KEY_ID]
   --aws-secret-access-key value                          AWS secret access key [$AWS_SECRET_ACCESS_KEY]
   --aws-hosted-zone-id value                             AWS hosted zone ID [$AWS_HOSTED_ZONE_ID]
   --debug-mode                                           Run in debug mode (uses staging Let's Encrypt server) (default: false) [$DEBUG_MODE]
   --help, -h                                             show help
   --version, -v                                          print the version
```
