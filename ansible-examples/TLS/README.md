# ydb-ca-nodes.txt

## Usage
This file is used by script `ydb-ca-update.sh`. Fill this file with list of your nodes and execute script `python3 ydb-ca-update.sh`. Script will generate set of self-signed certificates for every node in the file. These certificates will be placed in folder:
`CA/certs/YYYY-MM-DD_HH-mm-SS/<inventory hostname>`.

The first name on each line is the directory name and the certificate DNS SAN. It must be the full inventory hostname, exactly as that host is written in the inventory, including the domain. Ansible looks up `ydb_tls_dir/<inventory hostname>/node.crt`. The label before the first dot is not unique across zones, so it is not used as a directory name.

Additional names on the same line, separated by spaces, are extra DNS SANs. They do not change the directory name.

## Structure of file
\<inventory hostname>
\<inventory hostname> \<extra DNS name>
