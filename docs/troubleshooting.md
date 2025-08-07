# Troubleshooting Guide

## Overview

This guide covers common issues you may encounter when using the BadgerMaps CLI and provides solutions to resolve them.

## Common Issues

### Database Connection Issues

#### Issue: Database Connection Failed

**Symptoms:**
- Error: `connection refused`
- Error: `authentication failed`
- Error: `database does not exist`

**Solutions:**

1. **Check Database Server**
   ```bash
   # PostgreSQL
   sudo systemctl status postgresql
   
   # SQL Server
   sudo systemctl status mssql-server
   ```

2. **Verify Connection Parameters**
   ```bash
   # Check your .env file
   cat .env
   
   # Test connection manually
   psql -h localhost -U badger_user -d badgermaps
   ```

3. **Check Firewall Settings**
   ```bash
   # Check if port is open
   telnet localhost 5432  # PostgreSQL
   telnet localhost 1433  # SQL Server
   ```

#### Issue: SQLite3 File Permissions

**Symptoms:**
- Error: `unable to open database file`
- Error: `permission denied`

**Solutions:**

```bash
# Check file permissions
ls -la badgersync.db

# Fix permissions
chmod 644 badgersync.db
chmod 755 .

# Check disk space
df -h .
```

### API Connection Issues

#### Issue: API Authentication Failed

**Symptoms:**
- Error: `401 Unauthorized`
- Error: `invalid API key`

**Solutions:**

1. **Verify API Key**
   ```bash
   # Check API key in .env file
   grep BADGERMAPS_API_KEY .env
   
   # Test API key manually
   curl -H "Authorization: Bearer YOUR_API_KEY" \
        https://api.badgermapping.com/v1/customers/
   ```

2. **Check API Key Format**
   - Ensure no extra spaces or characters
   - Verify the key is complete
   - Check if the key has expired

#### Issue: API Rate Limiting

**Symptoms:**
- Error: `429 Too Many Requests`
- Slow response times
- Intermittent failures

**Solutions:**

1. **Implement Backoff**
   ```bash
   # The CLI automatically handles rate limiting
   # Wait and retry the operation
   ```

2. **Check API Usage**
   - Monitor your API usage limits
   - Reduce request frequency
   - Contact BadgerMaps support if needed

#### Issue: Network Connectivity

**Symptoms:**
- Error: `connection timeout`
- Error: `no route to host`

**Solutions:**

1. **Check Internet Connection**
   ```bash
   ping api.badgermapping.com
   curl -I https://api.badgermapping.com/v1/
   ```

2. **Check Proxy Settings**
   ```bash
   # Set proxy if needed
   export HTTP_PROXY=http://proxy.company.com:8080
   export HTTPS_PROXY=http://proxy.company.com:8080
   ```

### Configuration Issues

#### Issue: Environment Variables Not Loaded

**Symptoms:**
- Error: `configuration not found`
- Default values being used

**Solutions:**

1. **Check .env File**
   ```bash
   # Verify .env file exists
   ls -la .env
   
   # Check file content
   cat .env
   ```

2. **Verify File Format**
   ```bash
   # Ensure proper format
   DB_TYPE=sqlite3
   DB_NAME=badgersync.db
   BADGERMAPS_API_KEY=your_api_key_here
   ```

3. **Check Working Directory**
   ```bash
   # Ensure you're in the correct directory
   pwd
   ls -la .env
   ```

#### Issue: Invalid Configuration

**Symptoms:**
- Error: `unsupported database type`
- Error: `invalid port number`

**Solutions:**

1. **Validate Configuration**
   ```bash
   # Test configuration
   ./badgersync test
   ```

2. **Check Variable Values**
   ```bash
   # Database type must be: sqlite3, postgres, mssql
   DB_TYPE=sqlite3
   
   # Port must be numeric
   DB_PORT=5432
   ```

### Data Synchronization Issues

#### Issue: Pull Operation Fails

**Symptoms:**
- Error: `failed to pull data`
- Partial data synchronization

**Solutions:**

1. **Check API Connectivity**
   ```bash
   ./badgersync test
   ```

2. **Verify Database Schema**
   ```bash
   ./badgersync utils create-tables
   ```

3. **Check Logs**
   ```bash
   tail -f badgersync.log
   ```

#### Issue: Push Operation Fails

**Symptoms:**
- Error: `failed to push data`
- Data not updated in API

**Solutions:**

1. **Check API Permissions**
   - Verify API key has write permissions
   - Check if endpoint supports write operations

2. **Validate Data Format**
   ```bash
   # Check data in database
   sqlite3 badgersync.db "SELECT * FROM accounts LIMIT 5;"
   ```

3. **Check for Conflicts**
   - Verify data hasn't been modified elsewhere
   - Check for duplicate records

### Performance Issues

#### Issue: Slow Operations

**Symptoms:**
- Long execution times
- High memory usage
- Database locks

**Solutions:**

1. **Database Optimization**
   ```bash
   # SQLite3 optimization
   sqlite3 badgersync.db "VACUUM;"
   sqlite3 badgersync.db "ANALYZE;"
   ```

2. **Check Indexes**
   ```bash
   # Verify indexes exist
   sqlite3 badgersync.db ".indexes"
   ```

3. **Monitor Resources**
   ```bash
   # Check system resources
   top
   iostat
   ```

#### Issue: Memory Issues

**Symptoms:**
- Out of memory errors
- Slow performance with large datasets

**Solutions:**

1. **Process Data in Batches**
   - The CLI processes data in batches by default
   - Consider reducing batch size if needed

2. **Optimize Database Queries**
   - Use appropriate indexes
   - Limit result sets

3. **Monitor Memory Usage**
   ```bash
   # Check memory usage
   ps aux | grep badgersync
   ```

## Debug Mode

Enable debug logging to get detailed information:

```bash
# Set debug level in .env file
LOG_LEVEL=debug

# Run command with debug output
./badgersync pull accounts
```

Debug mode will show:
- Environment variable loading
- Database connection details
- API request/response details
- SQL query execution
- Error stack traces

## Log Analysis

### Understanding Log Messages

```bash
# View recent logs
tail -50 badgersync.log

# Search for errors
grep -i error badgersync.log

# Search for specific operations
grep "pull accounts" badgersync.log
```

### Common Log Patterns

1. **Connection Success**
   ```
   Successfully connected to sqlite3 database
   ```

2. **API Success**
   ```
   Retrieved 4967 accounts from API
   ```

3. **Database Success**
   ```
   Successfully stored 4967 accounts using merge_accounts_basic
   ```

4. **Error Patterns**
   ```
   Failed to connect to database: connection refused
   Failed to get accounts from API: 401 Unauthorized
   ```

## Getting Help

### Before Asking for Help

1. **Check Documentation**
   - Review relevant documentation files
   - Check troubleshooting sections

2. **Enable Debug Logging**
   ```bash
   LOG_LEVEL=debug ./badgersync your-command
   ```

3. **Gather Information**
   - CLI version: `./badgersync --version`
   - Platform: `uname -a`
   - Go version: `go version`
   - Configuration (without sensitive data)

### Reporting Issues

When reporting issues, include:

1. **Environment Information**
   - Operating system and version
   - Go version
   - CLI version
   - Database type and version

2. **Configuration**
   - Database configuration (without passwords)
   - API configuration (without keys)
   - Log level setting

3. **Error Details**
   - Complete error message
   - Debug log output
   - Steps to reproduce

4. **Expected vs Actual Behavior**
   - What you expected to happen
   - What actually happened
   - Any error messages

### Support Channels

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general help
- **Documentation**: Check the docs folder for detailed guides

## Prevention

### Best Practices

1. **Regular Backups**
   ```bash
   # Backup database
   cp badgersync.db badgersync.db.backup
   ```

2. **Test Configuration**
   ```bash
   # Test before running operations
   ./badgersync test
   ```

3. **Monitor Logs**
   ```bash
   # Check logs regularly
   tail -f badgersync.log
   ```

4. **Update Regularly**
   - Keep CLI updated to latest version
   - Monitor for security updates

5. **Secure Configuration**
   ```bash
   # Set proper file permissions
   chmod 600 .env
   chmod 644 badgersync.log
   ```

### Maintenance

1. **Database Maintenance**
   ```bash
   # SQLite3 maintenance
   sqlite3 badgersync.db "VACUUM; ANALYZE;"
   ```

2. **Log Rotation**
   ```bash
   # Rotate log files
   mv badgersync.log badgersync.log.old
   touch badgersync.log
   ```

3. **Configuration Review**
   - Regularly review configuration
   - Update API keys when needed
   - Verify database connections 