# 📋 Logging Guide - Go Template Application

## 🔍 Overview

The Go Template application has a comprehensive logging system with support for multiple formats, levels, and integrations including **ELK Stack (Elasticsearch + Kibana)** for advanced log analysis and visualization.

## 🏗️ Current Logging Architecture

### **Technologies Used:**
- **Logrus**: Structured logging library
- **Gin Middleware**: HTTP request/response logging
- **Elasticsearch Integration**: Ready for ELK stack
- **JSON & Text Formats**: Support for both human-readable and machine-parseable logs

### **Key Features:**
✅ Multiple log levels (debug, info, warn, error)
✅ Structured logging with key-value pairs
✅ JSON format for log aggregation
✅ HTTP request logging
✅ Redis connection logging
✅ Application lifecycle logging
✅ ELK Stack integration ready

## 🚀 How to View Logs

### 1. **Console Output (Default)**
```bash
# Start application with default settings
./bin/go-template

# You'll see logs like:
{"level":"info","msg":"Redis connection established successfully","time":"2025-09-18T22:12:37+03:30"}
{"level":"info","msg":"Starting HTTP server","address":"localhost:8080","time":"2025-09-18T22:12:37+03:30"}
[GIN] 2025/09/18 - 22:12:39 | 200 | 20.543µs | 127.0.0.1 | GET "/health"
```

### 2. **JSON Format (For ELK/Kibana)**
```bash
# Perfect for log aggregation and analysis
LOG_FORMAT=json ./bin/go-template
```

### 3. **Text Format (Human Readable)**
```bash
# Colored output for terminal viewing
LOG_FORMAT=text ./bin/go-template
```

### 4. **Debug Level Logging**
```bash
# Get detailed application logs
LOG_LEVEL=debug LOG_FORMAT=json ./bin/go-template
```

### 5. **Save Logs to File**
```bash
# Redirect all logs to file
./bin/go-template > logs/app.log 2>&1

# Follow logs in real-time
./bin/go-template 2>&1 | tee logs/app.log

# Monitor existing log file
tail -f logs/app.log
```

## 📊 Log Analysis Commands

### **Filter Logs by Level**
```bash
# Find all error logs
grep '"level":"error"' logs/app.log

# Find all request logs
grep 'GIN' logs/app.log

# Count successful requests
grep 'GIN' logs/app.log | grep '200' | wc -l

# Find authentication failures
grep 'GIN' logs/app.log | grep '401'
```

### **Real-time Log Monitoring**
```bash
# Watch logs live
tail -f logs/app.log

# Watch only error logs
tail -f logs/app.log | grep error

# Watch with syntax highlighting
tail -f logs/app.log | jq '.'
```

## 🏗️ ELK Stack Integration (Kibana Dashboards)

### **Setup ELK Stack**

Yes, you're absolutely right! The application has **complete ELK stack integration** ready to use:

1. **Start ELK Stack with your application:**
```bash
# Use the provided ELK docker-compose
docker-compose -f docker-compose.elk.yml up -d
```

2. **Configure Environment for ELK:**
```bash
export ELK_ENABLED=true
export ELK_URLS=http://localhost:9200
export LOG_FORMAT=json
export LOG_LEVEL=info
```

3. **Access Kibana Dashboard:**
   - Open browser: `http://localhost:5601`
   - Create index pattern: `go-template-logs-*`
   - Start building dashboards and visualizations!

### **What You'll See in Kibana:**
- 📈 Request volume over time
- 🚨 Error rates and patterns
- 🔍 Response time analytics
- 👤 User activity tracking
- 🎯 Cache hit/miss ratios
- 📊 API endpoint performance

### **Example Kibana Queries:**
```json
# Find all authentication errors
{
  "query": {
    "bool": {
      "must": [
        {"match": {"level": "error"}},
        {"match": {"message": "authentication"}}
      ]
    }
  }
}

# API response time analysis
{
  "query": {
    "range": {
      "response_time": {
      "gte": 1000
      }
    }
  }
}
```

## 🔧 Configuration Options

### **Environment Variables:**
```bash
# Log Level
LOG_LEVEL=debug|info|warn|error

# Log Format
LOG_FORMAT=json|text

# Log Output
LOG_OUTPUT=stdout|stderr|file

# File Logging
LOG_FILE_PATH=./logs/app.log
LOG_MAX_SIZE_MB=100
LOG_MAX_BACKUPS=3
LOG_MAX_AGE_DAYS=28
LOG_COMPRESS=true

# Request Logging
LOG_REQUESTS=true
LOG_HEADERS=false
LOG_BODY=false

# ELK Integration
ELK_ENABLED=true
ELK_URLS=http://localhost:9200
ELK_INDEX_PREFIX=go-template
ELK_BATCH_SIZE=100
```

## 📝 Log Structure

### **Application Logs (JSON Format):**
```json
{
  "level": "info",
  "msg": "Redis connection established successfully",
  "time": "2025-09-18T22:12:37+03:30"
}

{
  "address": "localhost:8080",
  "level": "info",
  "msg": "Starting HTTP server",
  "time": "2025-09-18T22:12:37+03:30"
}
```

### **HTTP Request Logs:**
```
[GIN] 2025/09/18 - 22:12:39 | 200 | 20.543µs | 127.0.0.1 | GET "/health"
[GIN] 2025/09/18 - 22:12:39 | 401 | 911.323µs | 127.0.0.1 | POST "/api/v1/auth/login"
```

## 🧪 Practical Examples for Mid-Level Developers

### **Quick Start - 5 Minutes Setup**

#### 1. **Basic Logging Setup**
```bash
# Clone and setup
git clone <your-repo>
cd go-template
cp .env.example .env

# Start Redis (for caching)
redis-server

# Start application with debug logging
LOG_LEVEL=debug LOG_FORMAT=text make run
```

#### 2. **Test API and Watch Logs**
```bash
# Terminal 1: Run application
LOG_FORMAT=json make run

# Terminal 2: Make API calls and watch logs
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "dev@test.com", "password": "test123", "first_name": "Dev", "last_name": "Test"}'
```

**Expected Log Output:**
```json
{"level":"info","msg":"Redis connection established successfully","time":"2025-09-18T22:12:37+03:30"}
{"level":"info","msg":"Starting HTTP server","address":"localhost:8080","time":"2025-09-18T22:12:37+03:30"}
[GIN] 2025/09/18 - 22:12:39 | 201 | 35.546803ms | 127.0.0.1 | POST "/api/v1/auth/register"
```

### **Real-World Development Scenarios**

#### **Scenario 1: Debugging Authentication Issues**
```bash
# Enable debug logging for auth troubleshooting
LOG_LEVEL=debug LOG_FORMAT=json ./bin/go-template 2>&1 | grep -E "(auth|login|token)"

# Test failed login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "wrong@test.com", "password": "wrong"}'

# Watch for authentication errors in logs
tail -f logs/app.log | jq 'select(.level == "error" and (.msg | contains("auth")))'
```

#### **Scenario 2: Performance Monitoring**
```bash
# Monitor API response times
LOG_REQUESTS=true ./bin/go-template 2>&1 | grep "GIN" | awk '{print $6, $8}' | sort -nr

# Output example:
# 35.546803ms POST
# 1.249641ms GET
# 180.093µs GET
```

#### **Scenario 3: Cache Debugging**
```bash
# Enable cache-specific logging
LOG_LEVEL=debug ./bin/go-template 2>&1 | grep -E "(cache|redis)"

# Monitor cache operations
redis-cli MONITOR &
MONITOR_PID=$!

# Make requests to see cache behavior
curl "http://localhost:8080/api/v1/users/?page=1&limit=5" -H "Authorization: Bearer TOKEN"
curl "http://localhost:8080/api/v1/users/?page=1&limit=5" -H "Authorization: Bearer TOKEN"  # Should hit cache

kill $MONITOR_PID
```

### **Log Analysis Examples**

#### **Find All Errors in Last Hour**
```bash
# Using jq for JSON logs
cat logs/app.log | jq -r 'select(.level == "error" and (.time | fromdateiso8601) > (now - 3600))'

# Using grep for text logs
grep "ERROR" logs/app.log | grep "$(date '+%Y-%m-%d %H')"
```

#### **API Endpoint Performance Analysis**
```bash
# Extract response times by endpoint
grep "GIN" logs/app.log | \
  awk '{print $8, $6}' | \
  sort | \
  uniq -c | \
  sort -nr

# Output:
#  15 GET 250ms
#   8 POST 450ms
#   3 PUT 180ms
```

#### **Authentication Failure Patterns**
```bash
# Find failed login attempts
cat logs/app.log | jq -r 'select(.msg | contains("Invalid credentials")) | .time'

# Count by hour
grep "Invalid credentials" logs/app.log | \
  cut -d'T' -f2 | \
  cut -d':' -f1 | \
  sort | \
  uniq -c
```

### **ELK Stack Practical Setup**

#### **Step-by-Step ELK Integration**
```bash
# 1. Start ELK services
docker-compose -f docker-compose.elk.yml up -d

# 2. Wait for services (about 2 minutes)
until curl -s http://localhost:9200/_cluster/health | grep -q '"status":"green\|yellow"'; do
  echo "Waiting for Elasticsearch..."
  sleep 10
done

# 3. Configure application for ELK
export ELK_ENABLED=true
export LOG_FORMAT=json
export LOG_LEVEL=info

# 4. Start application
./bin/go-template

# 5. Generate some logs
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "elk@test.com", "password": "test123", "first_name": "ELK", "last_name": "Test"}'
```

#### **Kibana Dashboard Setup**
```bash
# Access Kibana
open http://localhost:5601

# Create index pattern in Kibana:
# 1. Go to Stack Management → Index Patterns
# 2. Create pattern: "go-template-logs-*"
# 3. Set timestamp field: "@timestamp"
# 4. Go to Discover to see logs
```

#### **Useful Kibana Queries**
```json
// Find all authentication errors
{
  "query": {
    "bool": {
      "must": [
        {"match": {"level": "error"}},
        {"wildcard": {"msg": "*auth*"}}
      ]
    }
  }
}

// Response time analysis (>1 second)
{
  "query": {
    "bool": {
      "must": [
        {"exists": {"field": "response_time"}},
        {"range": {"response_time": {"gte": "1s"}}}
      ]
    }
  }
}

// User activity by IP
{
  "aggs": {
    "ips": {
      "terms": {
        "field": "client_ip.keyword",
        "size": 10
      }
    }
  }
}
```

### **Production Debugging Workflow**

#### **1. Quick Health Check**
```bash
# Check application status
curl http://localhost:8080/health

# Check recent errors
tail -100 logs/app.log | jq 'select(.level == "error")'

# Check Redis connectivity
redis-cli ping
```

#### **2. Performance Investigation**
```bash
# Top slow endpoints
grep "GIN" logs/app.log | \
  awk '{if($6 ~ /[0-9]+ms/ && $6+0 > 100) print $8, $6}' | \
  sort | uniq -c | sort -nr

# Memory usage over time
grep "memory" logs/app.log | tail -20
```

#### **3. User Experience Issues**
```bash
# Failed requests by status code
grep "GIN" logs/app.log | \
  awk '{print $4}' | \
  sort | uniq -c | sort -nr

# Most active users (by requests)
grep "GIN" logs/app.log | \
  awk '{print $10}' | \
  sort | uniq -c | sort -nr | head -10
```

## 🎯 Best Practices

### **For Development:**
- Use `LOG_FORMAT=text` for console readability
- Set `LOG_LEVEL=debug` for detailed information
- Use `tail -f` to monitor logs in real-time
- Test logging by making API calls and watching output

### **For Production:**
- Use `LOG_FORMAT=json` for log aggregation
- Set `LOG_LEVEL=info` or `warn` to reduce noise
- Enable ELK stack for comprehensive monitoring
- Set up log rotation and retention policies
- Monitor disk space for log files

### **For Debugging:**
- Set `LOG_LEVEL=debug` temporarily
- Enable `LOG_REQUESTS=true` to trace request flow
- Use structured logging with key-value pairs
- Correlate logs with Redis operations using `redis-cli MONITOR`

### **For Team Development:**
- Standardize log formats across environments
- Document custom log fields and their meanings
- Create Kibana dashboards for common debugging scenarios
- Set up alerts for critical error patterns

## 🚨 Monitoring & Alerting

With ELK stack, you can set up alerts for:
- Error rate spikes
- Slow response times
- Authentication failures
- Cache miss rates
- Database connection issues

## 🔮 Future Enhancements

The logging system is designed to be extensible and supports:
- Custom log hooks
- Additional output destinations
- Metric collection integration
- Advanced filtering and parsing
- Distributed tracing integration

---

**Ready to start monitoring?**
1. Enable JSON logging: `LOG_FORMAT=json`
2. Start ELK stack: `docker-compose -f docker-compose.elk.yml up -d`
3. Open Kibana: `http://localhost:5601`
4. Build amazing dashboards! 📊