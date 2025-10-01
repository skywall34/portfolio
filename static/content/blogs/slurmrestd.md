# Setting Up SLURM REST API (slurmrestd) - HPC Series Part 3

Tags: HPC, SLURM, REST API, JWT, Authentication

*This is part 3 of the HPC/SLURM series. Read [Part 1: Beginner's Guide to HPCs](/blogs/hpc) and [Part 2: Setting Up SLURM Database](/blogs/slurmdb) for the foundational setup.*

## Why Use the SLURM REST API?

Now that you have a working SLURM cluster with job accounting, you'll want to provide programmatic access to your cluster. The SLURM REST API (`slurmrestd`) allows you to:

- **Submit and manage jobs** via HTTP requests instead of command-line tools
- **Build custom web interfaces** for users who aren't comfortable with terminal commands
- **Integrate with other systems** like dashboards, monitoring tools, or automated workflows
- **Enable remote access** to cluster functionality through standard REST endpoints

For my personal use case, it made it easier for users to interact with SLURM without having to use the login or control node. In addition, it allowed us to build internal tools such as CLIs to make the developer experience that much easier.

## References

- Official Documentation: https://slurm.schedmd.com/rest_quickstart.html
- JSON Support: https://slurm.schedmd.com/related_software.html#json

## Setting Up slurmrestd

Here's how to set up `slurmrestd` using systemd and `.deb` packages:

### Step 1: Install slurmrestd

Download and install the required `.deb` packages:

```bash
dpkg -i slurm-smd.deb slurm-smd-slurmrestd.deb
```

These packages should be built using the same process described in [Part 1](/blogs/hpc#install-packages).

### Step 2: Create Dedicated User for slurmrestd

For security purposes, create a dedicated system user to run the slurmrestd service:

```bash
sudo useradd -M -r -s /usr/sbin/nologin -U slurmrestd
```

**Flags explained:**
- `-M`: Don't create a home directory
- `-r`: Create a system account
- `-s /usr/sbin/nologin`: No login shell (security best practice)
- `-U`: Create a group with the same name as the user

### Step 3: Enable JWT Authentication

JWT (JSON Web Tokens) are required for secure API authentication. Edit `/etc/slurm/slurm.conf` to enable JWT:

```bash
AuthType=auth/jwt
```

**Important**: After changing slurm.conf, you must restart all SLURM services on all nodes for the change to take effect.

### Step 4: Configure systemd Service

Create a systemd service override to configure how slurmrestd runs:

```bash
sudo systemctl edit slurmrestd
```

This will open an editor. Add the following configuration:

```ini
[Service]
User=slurmrestd
Group=slurmrestd
ExecStart=
ExecStart=/usr/sbin/slurmrestd $SLURMRESTD_OPTIONS
Environment=SLURMRESTD_LISTEN=:6820
```

**Configuration explained:**
- `User` and `Group`: Run the service as the dedicated slurmrestd user
- `ExecStart=`: Clear the default ExecStart directive
- `ExecStart=/usr/sbin/slurmrestd...`: Set the new ExecStart with options
- `Environment=SLURMRESTD_LISTEN=:6820`: Listen on port 6820 (you can change this)

Save and exit the editor.

### Step 5: Start slurmrestd

Reload systemd to apply the changes and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable slurmrestd
sudo systemctl start slurmrestd
```

Verify the service is running:

```bash
sudo systemctl status slurmrestd
```

## Testing the REST API

### Step 6: Generate and Test JWT Tokens

Generate a JWT token for authentication:

```bash
unset SLURM_JWT; export $(scontrol token)
```

This creates a JWT token and exports it to the `SLURM_JWT` environment variable.

Test the API with `curl`:

```bash
curl -s -o "/tmp/curl.log" -k -vvvv -H X-SLURM-USER-TOKEN:$SLURM_JWT \
-X GET 'http://localhost:6820/slurm/v0.0.36/diag'
```

If successful, you should receive diagnostic information about your SLURM cluster in JSON format.

**Common API Endpoints:**
- `/slurm/v0.0.36/diag` - Cluster diagnostics
- `/slurm/v0.0.36/jobs` - List all jobs
- `/slurm/v0.0.36/nodes` - List all nodes
- `/slurm/v0.0.36/partitions` - List all partitions

### Step 7: Configure JWT Token Expiration

By default, JWT tokens expire after 1800 seconds (30 minutes). For longer-running operations, you can specify a custom lifespan:

```bash
unset SLURM_JWT; export $(scontrol token lifespan=3600)
```

This creates a token valid for 3600 seconds (1 hour).

**Security Note**: Longer token lifespans are convenient but reduce security. Balance convenience with your security requirements.

## Example: Submitting a Job via REST API

Here's a basic example of submitting a job through the API:

```bash
# Generate token
export $(scontrol token lifespan=1800)

# Submit job
curl -X POST http://localhost:6820/slurm/v0.0.36/job/submit \
  -H "X-SLURM-USER-TOKEN: $SLURM_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "job": {
      "name": "test_job",
      "partition": "gpu",
      "nodes": 1,
      "tasks": 1,
      "script": "#!/bin/bash\necho \"Hello from SLURM REST API\"\n"
    }
  }'
```

## Security Considerations

When exposing the SLURM REST API:

1. **Use HTTPS in production** - Always use TLS/SSL for API endpoints exposed to the network
2. **Implement rate limiting** - Prevent abuse by limiting API request rates
3. **Token management** - Store and rotate tokens securely
4. **Firewall rules** - Restrict API access to trusted networks/IPs
5. **Audit logging** - Monitor API usage for suspicious activity

## Wrapping Up

You now have a fully functional SLURM cluster with:
- Job scheduling and resource management ([Part 1](/blogs/hpc))
- Job accounting and usage tracking ([Part 2](/blogs/slurmdb))
- Programmatic REST API access (Part 3)

This setup allows you to build custom interfaces, automate workflows, and provide an easy-to-use platform for your users - exactly what we built for our data science team.