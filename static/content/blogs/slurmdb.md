# Setting Up SLURM Database (slurmdbd) - HPC Series Part 2

Tags: HPC, SLURM, Database, Accounting

*This is part 2 of the HPC/SLURM series. Read [Part 1: Beginner's Guide to HPCs](/blog/hpc) for an introduction to SLURM and cluster setup.*

## Why Do You Need SLURM Database?

After setting up your basic SLURM cluster, you'll quickly realize the need to track job history, resource usage, and user accounting. This is where `slurmdbd` (SLURM Database Daemon) comes in. It provides:

- **Job Accounting**: Track who ran what jobs and when
- **Resource Usage Statistics**: Monitor CPU, GPU, and memory utilization over time
- **Fair-share Scheduling**: Prioritize jobs based on historical usage
- **Reporting**: Generate reports for billing, auditing, and resource planning

The SLURM database can be deployed either on the control node (simpler for small clusters) or on a separate database server (recommended for larger deployments).

For detailed information on separate node deployment, see: https://wiki.fysik.dtu.dk/Niflheim_system/Slurm_database/

## Deploying slurmdbd on the Control Node

### Step 1: Install and Configure MariaDB

First, install the MariaDB server and client packages:

```bash
sudo apt-get install mariadb-server mariadb-client
```

Secure the MariaDB installation and set a root password:

```bash
sudo mysql_secure_installation
```

This interactive script will guide you through:
- Setting a root password
- Removing anonymous users
- Disallowing root login remotely
- Removing test databases

Next, create the SLURM accounting database and user. Log into MariaDB:

```bash
sudo mysql -u root -p
```

Run the following SQL commands to set up the database:

```sql
CREATE DATABASE slurm_acct_db;
CREATE USER 'slurm'@'localhost' IDENTIFIED BY 'password';
GRANT ALL ON slurm_acct_db.* TO 'slurm'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

**Important**: Replace `'password'` with a strong, unique password for production use.

### Step 2: Configure SLURMDBD

Install the SLURM database daemon package:

```bash
sudo apt install slurmdbd
```

Create and edit the slurmdbd configuration file at `/etc/slurm/slurmdbd.conf`:

```bash
DbdHost=control-node
DbdPort=6819
StorageType=accounting_storage/mysql
StorageHost=localhost
StoragePass=password
StorageUser=slurm
StorageLoc=slurm_acct_db
```

**Configuration Parameters Explained:**
- `DbdHost`: The hostname of the server running slurmdbd (use your control node's hostname)
- `DbdPort`: Port for slurmdbd communication (default: 6819)
- `StorageType`: Backend storage type (MySQL/MariaDB)
- `StorageHost`: Database server location (localhost if on same node)
- `StoragePass`: Password for the slurm database user
- `StorageUser`: Database username
- `StorageLoc`: Name of the SLURM accounting database

Set proper permissions on the configuration file:

```bash
sudo chmod 600 /etc/slurm/slurmdbd.conf
sudo chown slurm:slurm /etc/slurm/slurmdbd.conf
```

Enable and start the SLURM database daemon:

```bash
sudo systemctl enable slurmdbd
sudo systemctl start slurmdbd
```

Verify the service is running:

```bash
sudo systemctl status slurmdbd
```

### Step 3: Configure SLURM to Use SLURMDBD

Update the SLURM configuration file `/etc/slurm/slurm.conf` on the control node to enable accounting. Add these lines to your slurm.conf:

```bash
AccountingStorageType=accounting_storage/slurmdbd
AccountingStorageHost=control-node
AccountingStoragePort=6819
```

Replace `control-node` with your actual control node hostname.

Restart SLURM services on the control node to apply the changes:

```bash
sudo systemctl restart slurmctld
```

## Verifying Your Setup

After configuration, verify that accounting is working properly:

```bash
# Check cluster status
sacctmgr show cluster

# Add your cluster if it doesn't exist
sacctmgr add cluster <cluster-name>

# View accounting data
sacct

# Check job history
sacct -a --format=JobID,JobName,User,Partition,State,Start,End,Elapsed
```

## Next Steps

Now that you have job accounting set up, you can:
- Set up user accounts and associations with `sacctmgr`
- Configure fair-share scheduling policies
- Generate usage reports for resource planning
- Implement job limits per user or partition

Continue to [Part 3: Setting Up SLURM REST API](/blog/slurmrestd) to learn how to interact with your cluster programmatically.