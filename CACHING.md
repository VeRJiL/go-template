# 🚀 Caching Implementation Guide

## 📖 Overview

This guide explains how Redis caching is implemented in the Go Template application, providing mid-level developers with everything they need to understand, extend, and maintain the caching system.

## 🏗️ Architecture Overview

### **Cache-First Strategy**
```
API Request → Check Cache → Cache Hit? → Return Data
                     ↓
                Cache Miss → Database Query → Store in Cache → Return Data
```

### **Automatic Invalidation**
```
Data Mutation (Create/Update/Delete) → Invalidate Cache → Next Request Rebuilds Cache
```

## 🔧 Implementation Details

### **Core Components**

#### 1. **Cache Repository Interface** (`internal/domain/repositories/user_repository.go:20-30`)
```go
type UserCacheRepository interface {
    Set(ctx context.Context, key string, user *entities.User) error
    Get(ctx context.Context, key string) (*entities.User, error)
    Delete(ctx context.Context, key string) error
    SetJSON(ctx context.Context, key string, data interface{}) error
    GetJSON(ctx context.Context, key string, dest interface{}) error
    DeletePattern(ctx context.Context, pattern string) error
    SetSession(ctx context.Context, token string, userID uuid.UUID) error
    GetSession(ctx context.Context, token string) (uuid.UUID, error)
    DeleteSession(ctx context.Context, token string) error
}
```

#### 2. **Redis Implementation** (`internal/database/redis/user_cache_repository.go`)
```go
type userCacheRepository struct {
    client *redis.Client
    ttl    time.Duration  // 24 hours default
}

func (r *userCacheRepository) SetJSON(ctx context.Context, key string, data interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal data: %w", err)
    }
    return r.client.Set(ctx, key, jsonData, r.ttl).Err()
}

func (r *userCacheRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
    data, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return fmt.Errorf("data not found in cache")
    }
    if err != nil {
        return fmt.Errorf("failed to get data from cache: %w", err)
    }
    return json.Unmarshal([]byte(data), dest)
}
```

#### 3. **Service Layer Integration** (`internal/domain/services/user_service.go:110-131`)
```go
func (s *UserService) List(ctx context.Context, offset, limit int) ([]*entities.User, int, error) {
    cacheKey := s.generateListCacheKey(offset, limit)

    // Try cache first
    if s.userCacheRepo != nil {
        if cachedData := s.getCachedUserList(ctx, cacheKey); cachedData != nil {
            return cachedData.Users, cachedData.Total, nil
        }
    }

    // Fallback to database
    users, total, err := s.userRepo.List(ctx, offset, limit)
    if err != nil {
        return nil, 0, err
    }

    // Cache the result
    if s.userCacheRepo != nil {
        s.cacheUserList(ctx, cacheKey, users, total)
    }

    return users, total, nil
}
```

### **Cache Key Generation**
```go
func (s *UserService) generateListCacheKey(offset, limit int) string {
    key := fmt.Sprintf("users:list:offset:%d:limit:%d", offset, limit)
    hash := md5.Sum([]byte(key))
    return fmt.Sprintf("users_list_%x", hash)
}
```

**Why MD5?**
- Consistent length regardless of parameters
- Avoids Redis key length limits
- Prevents special characters in keys

### **Cache Invalidation Strategy**
```go
func (s *UserService) Create(ctx context.Context, req *entities.CreateUserRequest) (*entities.User, error) {
    // ... create user logic ...

    s.invalidateUserListCache(ctx)  // ← Invalidate after data change
    return user, nil
}

func (s *UserService) invalidateUserListCache(ctx context.Context) {
    if s.userCacheRepo == nil {
        return
    }
    s.userCacheRepo.DeletePattern(ctx, "users_list_*")  // ← Remove all list caches
}
```

## 🛠️ Configuration

### **Environment Variables**
```bash
# Redis Connection
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Connection Pool
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=3

# Timeouts
REDIS_DIAL_TIMEOUT=5
REDIS_READ_TIMEOUT=3
REDIS_WRITE_TIMEOUT=3

# Cache TTL (Time To Live)
CACHE_USER_TTL=3600        # 1 hour
CACHE_SESSION_TTL=86400    # 24 hours
CACHE_DEFAULT_TTL=1800     # 30 minutes
```

### **Application Setup** (`internal/app/app.go:67-90`)
```go
a.redisClient = redis.NewClient(&redis.Options{
    Addr:         a.config.Redis.Host + ":" + a.config.Redis.Port,
    Password:     a.config.Redis.Password,
    DB:           a.config.Redis.DB,
    PoolSize:     a.config.Redis.PoolSize,
    MinIdleConns: a.config.Redis.MinIdleConns,
    DialTimeout:  a.config.Redis.DialTimeout,
    ReadTimeout:  a.config.Redis.ReadTimeout,
    WriteTimeout: a.config.Redis.WriteTimeout,
})

// Test connection with graceful fallback
if err := a.redisClient.Ping(ctx).Err(); err != nil {
    a.logger.Warn("Redis connection failed, caching will be disabled", "error", err)
    a.redisClient = nil  // ← Application continues without cache
}
```

## 🧪 Testing Cache Behavior

### **Manual Testing**

#### 1. **Start Redis and Application**
```bash
# Start Redis
redis-server

# Start application
make run
```

#### 2. **Test Cache Population**
```bash
# First request (hits database, populates cache)
curl "http://localhost:8080/api/v1/users/?page=1&limit=5" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Check Redis for cache entry
redis-cli KEYS "users_list_*"
# Output: users_list_b4d5dc281b8f77ea8b7117dd83f7452c
```

#### 3. **Test Cache Hit**
```bash
# Second request (served from cache - should be faster)
time curl "http://localhost:8080/api/v1/users/?page=1&limit=5" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 4. **Test Cache Invalidation**
```bash
# Create new user (should invalidate cache)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@cache.com", "password": "test123", "first_name": "Test", "last_name": "User"}'

# Check if cache was cleared
redis-cli KEYS "users_list_*"
# Output: (empty)

# Next request will rebuild cache
curl "http://localhost:8080/api/v1/users/?page=1&limit=5" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### **Redis CLI Commands for Debugging**
```bash
# Monitor all Redis operations
redis-cli MONITOR

# List all cache keys
redis-cli KEYS "*"

# Get cache content
redis-cli GET "users_list_abc123"

# Check TTL
redis-cli TTL "users_list_abc123"

# Clear all cache
redis-cli FLUSHALL
```

## 🚀 Adding Caching to New Endpoints

### **Step-by-Step Guide**

#### 1. **Define Cache Methods in Repository Interface**
```go
// In internal/domain/repositories/your_repository.go
type YourCacheRepository interface {
    SetYourData(ctx context.Context, key string, data *YourEntity) error
    GetYourData(ctx context.Context, key string) (*YourEntity, error)
    DeleteYourDataPattern(ctx context.Context, pattern string) error
}
```

#### 2. **Implement Cache Methods**
```go
// In internal/database/redis/your_cache_repository.go
func (r *yourCacheRepository) SetYourData(ctx context.Context, key string, data *YourEntity) error {
    return r.SetJSON(ctx, key, data)  // ← Reuse existing JSON methods
}

func (r *yourCacheRepository) GetYourData(ctx context.Context, key string) (*YourEntity, error) {
    var data YourEntity
    if err := r.GetJSON(ctx, key, &data); err != nil {
        return nil, err
    }
    return &data, nil
}
```

#### 3. **Update Service Layer**
```go
// In internal/domain/services/your_service.go
func (s *YourService) GetData(ctx context.Context, id string) (*YourEntity, error) {
    cacheKey := fmt.Sprintf("your_data_%s", id)

    // Check cache first
    if s.cacheRepo != nil {
        if cached, err := s.cacheRepo.GetYourData(ctx, cacheKey); err == nil {
            return cached, nil
        }
    }

    // Fallback to database
    data, err := s.repository.GetData(ctx, id)
    if err != nil {
        return nil, err
    }

    // Cache the result
    if s.cacheRepo != nil {
        s.cacheRepo.SetYourData(ctx, cacheKey, data)
    }

    return data, nil
}
```

#### 4. **Add Cache Invalidation**
```go
func (s *YourService) UpdateData(ctx context.Context, id string, updates *UpdateRequest) error {
    // Update database
    err := s.repository.Update(ctx, id, updates)
    if err != nil {
        return err
    }

    // Invalidate specific cache entry
    if s.cacheRepo != nil {
        cacheKey := fmt.Sprintf("your_data_%s", id)
        s.cacheRepo.Delete(ctx, cacheKey)
    }

    return nil
}
```

## 📊 Performance Monitoring

### **Cache Hit Ratio Tracking**
```go
// Add metrics to your service
type CacheMetrics struct {
    Hits   int64
    Misses int64
}

func (s *YourService) GetWithMetrics(ctx context.Context, key string) (*Data, error) {
    if cached := s.getFromCache(ctx, key); cached != nil {
        atomic.AddInt64(&s.metrics.Hits, 1)
        return cached, nil
    }

    atomic.AddInt64(&s.metrics.Misses, 1)
    // ... fetch from database
}
```

### **Logging Cache Operations**
```go
func (s *UserService) getCachedUserList(ctx context.Context, cacheKey string) *UserListCacheData {
    var cacheData UserListCacheData
    if err := s.userCacheRepo.GetJSON(ctx, cacheKey, &cacheData); err != nil {
        s.logger.Debug("Cache miss", "key", cacheKey, "error", err)
        return nil
    }
    s.logger.Debug("Cache hit", "key", cacheKey, "users_count", len(cacheData.Users))
    return &cacheData
}
```

## 🔧 Troubleshooting

### **Common Issues**

#### 1. **Cache Not Working**
```bash
# Check Redis connection
redis-cli ping
# Expected: PONG

# Check application logs
grep "Redis connection" logs/app.log
```

#### 2. **Stale Data in Cache**
```bash
# Force cache refresh by clearing
redis-cli DEL "users_list_*"

# Or restart application to reconnect to Redis
```

#### 3. **Memory Usage**
```bash
# Check Redis memory usage
redis-cli INFO memory

# Set max memory policy
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

#### 4. **Performance Issues**
```bash
# Monitor Redis operations
redis-cli --latency

# Check slow queries
redis-cli SLOWLOG GET 10
```

## 📋 Best Practices

### **Do's**
✅ Always check if cache repository is available before using
✅ Use structured cache keys with consistent patterns
✅ Implement graceful degradation when Redis is unavailable
✅ Invalidate cache on all data mutations
✅ Use appropriate TTL values for different data types
✅ Monitor cache hit rates and performance

### **Don'ts**
❌ Don't cache frequently changing data
❌ Don't store sensitive information in cache
❌ Don't rely solely on cache without database fallback
❌ Don't use cache for user-specific data without proper isolation
❌ Don't ignore cache errors silently

### **Cache Key Patterns**
```bash
# Good patterns
users_list_{hash}           # List queries with pagination
user_profile_{user_id}      # Individual user data
session_{token_hash}        # Session data

# Bad patterns
user_data                   # Too generic
users_2024_01_01           # Date-specific without business logic
temp_cache_123             # Temporary or unclear purpose
```

## 🎯 Production Considerations

### **Redis Configuration**
```bash
# Persistence
save 900 1      # Save if at least 1 key changed in 900 seconds
save 300 10     # Save if at least 10 keys changed in 300 seconds
save 60 10000   # Save if at least 10000 keys changed in 60 seconds

# Memory Management
maxmemory 2gb
maxmemory-policy allkeys-lru

# Security
requirepass your_strong_password
```

### **Monitoring Setup**
- Redis memory usage alerts
- Cache hit rate monitoring
- Connection pool exhaustion alerts
- Slow query detection

### **Disaster Recovery**
- Regular Redis backups
- Cache warming strategies
- Circuit breaker patterns for cache failures

---

This caching implementation provides a solid foundation for building high-performance applications with automatic cache management and graceful degradation.