# Querying the database

The following section provides some sample queries you can use with the data
collected by the Inventory system.

## AWS VPCs Per Region

The following query will give you the number of VPCs per Region.

``` sql
SELECT
        v.region_name AS region_name,
        v.account_id AS account_id,
        COUNT(v.id) AS total
FROM aws_vpc AS v
GROUP BY (v.region_name, v.account_id)
ORDER BY total DESC;
```

## AWS Instances Grouped By Type

Returns the AWS instances grouped by type.

``` sql
SELECT
        instance_type,
        account_id,
        COUNT(id) AS total
FROM aws_instance
GROUP BY (instance_type, account_id) ORDER BY total DESC;
```

## AWS Instances Grouped By Region and VPC

The following query returns the list of instances grouped by Region and VPC.

``` sql
SELECT
        v.name AS vpc_name,
        v.region_name AS region_name,
        v.account_id AS account_id,
        COUNT(i.id) AS instances
FROM aws_vpc AS v
INNER JOIN aws_instance AS i ON v.vpc_id = i.vpc_id
GROUP BY (v.name, v.region_name, v.account_id)
ORDER BY instances DESC;
```

## AWS EC2 Instances Grouped By Arch

``` sql
SELECT
        arch,
        COUNT(id) AS total
FROM aws_instance
GROUP BY arch ORDER BY total DESC;
```

Sample output:

``` text
┌────────┬───────┐
│  arch  │ total │
├────────┼───────┤
│ x86_64 │    85 │
└────────┴───────┘
(1 row)

Time: 2.631 ms
```

## AWS EC2 Instances Uptime

Filter out EC2 instances with an uptime of more than 30 days.

``` sql
SELECT * FROM aws_instance WHERE launch_time < NOW() - INTERVAL '30 days';
```

## AWS Public IP Addresses Per Region

The following query returns the total number of public IP addresses per AWS Region.

``` sql
SELECT
        region_name,
        account_id,
        COUNT(id) AS public_ip_addresses
FROM aws_net_interface
WHERE public_ip_address <> ''
GROUP BY region_name, account_id ORDER BY public_ip_addresses DESC;
```

## AWS Load Balancers and Network Interfaces

The following query will return the Elastic Load Balancers, along with their
private and public IPv4 addresses, by joining the ELB and NetworkInterface using
the link table.

``` sql
SELECT
        lb.id AS lb_id,
        lb.dns_name AS dns,
        lb.region_name AS region,
        lb.account_id AS account_id,
        lb.type AS lb_type,
        ni.private_ip_address AS priv_ip_addr,
        ni.public_ip_address AS pub_ip_addr
FROM aws_loadbalancer AS lb
INNER JOIN l_aws_lb_to_net_interface AS link ON lb.id = link.lb_id
INNER JOIN aws_net_interface AS ni ON ni.id = link.ni_id;
```

## AWS EC2 Instances with Network Interfaces

The following query will join the EC2 instances with the Elastic Network
Interfaces.

``` sql
SELECT
        i.*,
        ni.private_ip_address,
        ni.public_ip_address,
        ni.mac_address
FROM aws_instance AS i
INNER JOIN aws_net_interface AS ni ON i.instance_id = ni.instance_id AND i.account_id = ni.account_id
```

## AWS EC2 Instances which are using unknown CloudProfile image

The following query will return the set of EC2 instances, which are using
images, which are not defined in the respective CloudProfile for the shoot they
belong to.

``` sql
SELECT DISTINCT
       i.*,
       s.name AS shoot_name,
       s.project_name AS project_name
FROM aws_instance AS i
INNER JOIN g_machine AS m ON i.name = m.name
INNER JOIN g_shoot AS s ON m.namespace = s.technical_id
LEFT JOIN g_cloud_profile_aws_image AS cpaw ON s.cloud_profile = cpaw.cloud_profile_name AND i.image_id = cpaw.ami
WHERE cpaw.ami IS NULL;
```

## Shoots Grouped by Cloud Profile

The following query will give you the shoots grouped by cloud profile.

``` sql
SELECT
        s.cloud_profile AS cloud_profile,
        COUNT(id) AS total
FROM g_shoot AS s
GROUP BY s.cloud_profile
ORDER BY total DESC;
```

Sample result.

``` text
┌────────────────────┬───────┐
│   cloud_profile    │ total │
├────────────────────┼───────┤
│ aws                │    97 │
│ gcp                │    39 │
│ az                 │    20 │
│ converged-cloud-cp │    12 │
│ alicloud           │    12 │
│ converged-cloud    │     4 │
│ ironcore           │     2 │
└────────────────────┴───────┘
(7 rows)

Time: 0.725 ms
```

## Top 10 Projects by Shoot Number

The following query will give you the top 10 projects with shoots.

``` sql
SELECT
        s.project_name AS project_name,
        p.owner AS project_owner,
        COUNT(s.id) AS total
FROM g_shoot AS s
INNER JOIN g_project AS p ON s.project_name = p.name
GROUP BY s.project_name, p.owner
ORDER BY total DESC
LIMIT 10;
```

## Group Number of Shoots per Seed

The following query will group the number of shoots per seed cluster.

``` sql
SELECT
        seed.name,
        COUNT(shoot.id) AS total
FROM g_seed AS seed
INNER JOIN g_shoot AS shoot ON seed.name = shoot.seed_name
GROUP BY seed.name
ORDER BY total DESC;
```

Sample result.

``` text
┌────────────────────────┬───────┐
│          name          │ total │
├────────────────────────┼───────┤
│ aws-ha                 │   108 │
│ az-ha                  │    30 │
│ gcp-ha                 │    23 │
│ gcp-cilium             │    22 │
│ ali-ha                 │    16 │
│ cc-ha                  │    12 │
│ soil-gcp-regional      │    11 │
│ soil-ccee-cp           │     1 │
│ soil-cc-ha             │     1 │
│ soil-kubernikus-cp     │     1 │
│ soil-kubernikus-eu-de1 │     1 │
└────────────────────────┴───────┘
(11 rows)

Time: 1.440 ms
```

## Number of Shoots per User

The following query will return the number of shoots grouped by the user, who
created them.

``` sql
SELECT
        s.created_by,
        COUNT(s.id) AS total
FROM g_shoot AS s
GROUP BY s.created_by
ORDER BY total DESC;
```

## Match Gardener Shoots with AWS VPCs

The following query will match the Gardener Shoots with the AWS VPCs.

``` sql
SELECT
        s.name AS shoot_name,
        s.namespace AS shoot_ns,
        s.technical_id AS shoot_tech_id,
        is_hibernated::text,
        s.project_name,

        v.vpc_id AS aws_vpc_id,
        v.account_id AS aws_account_id,
        v.region_name AS aws_region
FROM g_shoot AS s
INNER JOIN aws_vpc AS v ON s.technical_id = v.name;
```

## Match AWS VPCs with Gardener Shoots

This query is similar to the `Match Gardener Shoots with AWS VPCs`, but slightly
different, as it allows us to find VPCs for which there is no corresponding
shoot.

``` sql
SELECT
        v.name AS vpc_name,
        v.region_name AS region_name,
        v.vpc_id AS vpc_id,
        v.account_id AS account_id,

        s.name AS shoot_name,
        s.technical_id as shoot_tech_id
FROM aws_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id;
```

## Find Leaked AWS VPCs

Using the query from `Match AWS VPCs with Gardener Shoots` we can filter out the
results for VPCs, which do not have a corresponding shoot in Gardener, e.g.

``` sql
SELECT
        v.name AS vpc_name,
        v.region_name AS region_name,
        v.vpc_id AS vpc_id,
        v.account_id AS account_id,
        v.is_default::text,

        s.name AS shoot_name,
        s.technical_id as shoot_tech_id
FROM aws_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE s.technical_id is NULL;
```

We can further filter the results by `v.is_default` to exclude the default VPCs.

## Match AWS EC2 Instance with Gardener Machine

The following query will match the AWS EC2 instances with Gardener Machine
objects.

``` sql
SELECT
        i.name AS instance_name,
        i.instance_id AS instance_id,
        i.instance_type AS instance_type,
        i.state AS instance_state,
        i.vpc_id AS vpc_id,
        i.account_id AS account_id,

        m.name AS machine_name,
        m.provider_id AS machine_provider_id
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name;
```

## Match AWS EC2 Instance with Machine, VPC and Shoot

``` sql
SELECT
        i.name AS inst_name,
        i.instance_id AS inst_id,
        i.instance_type AS inst_type,
        i.state AS inst_state,
        i.account_id AS account_id,

        m.name AS machine_name,
        m.provider_id AS provider_id,

        v.name AS vpc_name,
        v.vpc_id AS vpc_id,
        v.region_name AS region,

        s.name AS shoot_name,
        s.project_name AS project
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN aws_vpc AS v ON i.vpc_id = v.vpc_id AND i.account_id = v.account_id
LEFT JOIN g_shoot AS s ON v.name = s.technical_id;
```

## Find Leaked AWS EC2 Instances

Using the query from `Match AWS EC2 Instance with Machine, VPC and Shoot` as a
starting point we can filter out the results to get a list of EC2 Instances
which do not have a corresponding Gardener Machine.

``` sql
SELECT
        i.name AS inst_name,
        i.instance_id AS inst_id,
        i.instance_type AS inst_type,
        i.state AS inst_state,
        i.account_id,

        v.name AS vpc_name,
        v.vpc_id AS vpc_id,
        v.region_name AS region,

        m.name AS machine_name,
        m.provider_id AS provider_id,

        s.name AS shoot_name,
        s.project_name AS project
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN aws_vpc AS v ON i.vpc_id = v.vpc_id AND i.account_id = v.account_id
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE i.state <> 'terminated' AND m.name IS NULL;
```

## Find Leaked AWS S3 Buckets

The following query will report AWS S3 buckets which do not have a corresponding
Gardener `BackupBucket`.

``` sql
SELECT
        b.name,
        b.region_name,
        b.account_id
FROM aws_bucket AS b
LEFT JOIN g_backup_bucket AS gbb ON b.name = gbb.name
WHERE gbb.name IS NULL;
```
