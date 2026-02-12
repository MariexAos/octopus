# Short Link Service Design Document

## Overview

This document describes the design of a short link service based on Go/Gin framework, providing URL shortening and redirection capabilities with parameter merging, access analytics, and traffic source analysis.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Web Framework | Gin |
| Encoding Algorithm | Base32 |
| Cache | Redis |
| Bloom Filter | Redis Bloom Filter |
| Database | MySQL |
| Message Queue | RocketMQ |
| Data Warehouse | ClickHouse/Hive |

## 1. Application Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        Client[Client Browser]
    end

    subgraph "API Gateway Layer"
        Gateway[API Gateway/Nginx]
    end

    subgraph "Short Link Service Layer"
        Router[GIN Router]

        subgraph "Generation Module"
            GenAPI[Generate Short Link API]
            Encoder[Base32 Encoder]
            CollisionCheck[Collision Checker]
            BloomFilter[Redis Bloom Filter]
        end

        subgraph "Redirect Module"
            RedirectAPI[302 Redirect API]
            URLDecoder[URL Decoder]
            ParamsParser[Parameter Parser]
        end

        subgraph "Analytics Module"
            PVCounter[PV Counter]
            UVTracker[UV Tracker]
            SourceTracker[Source Tracker]
        end
    end

    subgraph "Storage Layer"
        Redis[Redis Cache]
        MySQL[MySQL Database]
    end

    subgraph "Message Queue Layer"
        RocketMQ[RocketMQ]
        Producer[Log Producer]
        Consumer[Log Consumer]
    end

    subgraph "Data Warehouse Layer"
        DataWarehouse[ClickHouse/Hive]
        Analytics[Analytics Service]
    end

    Client -->|HTTP Request| Gateway
    Gateway --> Router
    Router --> GenAPI
    Router --> RedirectAPI

    GenAPI --> Encoder
    Encoder --> CollisionCheck
    CollisionCheck --> BloomFilter
    CollisionCheck --> Redis
    CollisionCheck --> MySQL

    RedirectAPI --> URLDecoder
    URLDecoder --> Redis
    URLDecoder --> ParamsParser

    RedirectAPI -->|PV/UV/Source Analytics| PVCounter
    PVCounter --> UVTracker
    UVTracker --> SourceTracker

    PVCounter --> Producer
    SourceTracker --> Producer

    Producer --> RocketMQ
    RocketMQ --> Consumer
    Consumer --> DataWarehouse
    DataWarehouse --> Analytics
```

## 2. Short Link Generation Algorithm

```mermaid
flowchart TD
    Start([Start]) --> Input[Receive Original URL and Parameters]
    Input --> Validate[Validate URL Format]

    Validate -->|Invalid| Error1[Return Error]
    Validate -->|Valid| CheckCache{Exists in<br/>Redis Cache?}

    CheckCache -->|Yes| Return1([Return Existing Short Link])
    CheckCache -->|No| Hash[Calculate URL+Params Hash]

    Hash --> BloomCheck{Bloom Filter<br/>Collision Check?}

    BloomCheck -->|Possible Collision| CheckDB{Actually Exists<br/>in Database?}

    BloomCheck -->|No Collision| Length4{4 Chars Enough?<br/>>=4^32 Combinations}

    CheckDB -->|Yes| Return1
    CheckDB -->|No| Collision[Hash Collision Occurred]
    Collision --> BloomCheck

    Length4 -->|Yes| Encode4[Base32 Encode 4 Chars]
    Length4 -->|No| Length6{6 Chars Enough?}

    Encode4 --> FinalCheck{Final Collision Check?}

    Length6 -->|Yes| Encode6[Base32 Encode 6 Chars]
    Length6 -->|No| Error2[Capacity Limit Exceeded]

    Encode6 --> FinalCheck

    FinalCheck -->|Collision| Retry[Increment and Re-encode]
    FinalCheck -->|No Collision| SaveDB[Save to MySQL]

    Retry --> FinalCheck

    SaveDB --> SaveCache[Write to Redis Cache]
    SaveCache --> UpdateBloom[Update Bloom Filter]
    UpdateBloom --> Return2([Return Generated Short Link])
```

### Generation Algorithm Description

1. **Input Validation**: Validate the original URL format
2. **Cache Check**: Check if a short link for this URL already exists in Redis
3. **Hash Calculation**: Calculate hash of URL + parameters
4. **Bloom Filter Check**: Use Redis Bloom Filter for fast collision detection
5. **Database Verification**: Query database when Bloom Filter indicates possible collision
6. **Base32 Encoding**: Generate 4-6 character short link from hash value
7. **Collision Handling**: Increment hash value and re-encode when collision detected
8. **Persistence**: Save to MySQL and Redis, update Bloom Filter

## 3. Short Link Redirect Algorithm

```mermaid
flowchart TD
    Start([User Visits Short Link]) --> Parse[Parse Short Code]
    Parse --> Extract[Extract Short Key and Parameters]

    Extract --> CheckCache{Exists in<br/>Redis Cache?}

    CheckCache -->|Yes| GetURL[Get Original URL]
    CheckCache -->|No| CheckDB{Exists in<br/>MySQL Database?}

    CheckDB -->|Yes| GetURL
    CheckDB -->|No| Error404[Return 404 Error]

    GetURL --> ParseParams[Parse URL Placeholder Parameters]
    ParseParams --> MergeParams[Merge Query Parameters]

    MergeParams --> BuildURL[Build Complete Target URL]

    BuildURL --> CollectPV[Record PV Count]
    BuildURL --> CollectUV[Record UV Count<br/>Based on Cookie/IP]
    BuildURL --> CollectSource[Record Source Info<br/>Referer/UserAgent]

    CollectPV --> SendMQ{Send to MQ?}
    CollectUV --> SendMQ
    CollectSource --> SendMQ

    SendMQ -->|Yes| AsyncLog[Async Write to RocketMQ]
    SendMQ -->|No| LocalLog[Fallback to Local Log]

    AsyncLog --> Redirect
    LocalLog --> Redirect

    Redirect([Return 302 Redirect<br/>Location: Target URL]) --> End([End])

    subgraph "Async Processing Flow"
        MQConsumer[RocketMQ Consumer] --> Batch[Batch Processing]
        Batch --> DW[Write to Data Warehouse]
        DW --> Analytics[Statistical Analysis]
        Analytics -->|PV/UV/Hot Topics| Report[Report Display]
    end
```

### Redirect Algorithm Description

1. **URL Parsing**: Parse short link key and attached parameters
2. **Query Original URL**: Get from Redis first, fallback to MySQL if cache miss
3. **Parameter Processing**: Parse URL placeholders, merge query parameters
4. **Analytics Collection**: Collect PV, UV and source information
5. **Async Logging**: Send analytics data to RocketMQ
6. **302 Redirect**: Return HTTP 302 status code with Location header

## 4. Base32 Encoding

### Character Set

Base32 uses character set: `ABCDEFGHIJKLMNOPQRSTUVWXYZ234567`

### Capacity Calculation

| Short Link Length | Combinations | Description |
|-------------------|--------------|-------------|
| 4 characters | 32^4 = 1,048,576 | ~1 million, suitable for small scale |
| 5 characters | 32^5 = 33,554,432 | ~33 million, medium scale |
| 6 characters | 32^6 = 1,073,741,824 | ~1 billion, large scale |

### Encoding Strategy

- Prefer 4-character encoding
- Auto-upgrade to 5 characters when 4-character collision rate is high or capacity insufficient
- Use 6 characters in extreme cases

## 5. Project Structure

```
shortlink/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── handler/
│   │   ├── generate.go       # Generate short link handler
│   │   └── redirect.go       # Redirect handler
│   ├── service/
│   │   ├── shortlink.go      # Short link service
│   │   ├── bloom.go          # Bloom Filter service
│   │   └── analytics.go      # Analytics service
│   ├── repository/
│   │   ├── redis.go          # Redis operations
│   │   └── mysql.go          # MySQL operations
│   ├── mq/
│   │   ├── producer.go       # RocketMQ producer
│   │   └── consumer.go       # RocketMQ consumer
│   ├── encoder/
│   │   └── base32.go         # Base32 encoder
│   └── model/
│       ├── shortlink.go      # Short link data model
│       └── access_log.go     # Access log model
├── pkg/
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── middleware/
│       ├── logger.go         # Logging middleware
│       └── recovery.go       # Panic recovery
├── go.mod
└── go.sum
```

## 6. API Design

### 1. Generate Short Link

**Request**

```http
POST /api/v1/shortlink/generate
Content-Type: application/json

{
  "url": "https://example.com/path",
  "params": {
    "utm_source": "wechat",
    "campaign": "promo123"
  }
}
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "short_link": "https://s.example.com/AbCd",
    "short_code": "AbCd",
    "original_url": "https://example.com/path",
    "expire_at": "2025-12-31T23:59:59Z"
  }
}
```

### 2. Short Link Redirect

**Request**

```http
GET /AbCd?param1=value1
```

**Response**

```http
HTTP/1.1 302 Found
Location: https://example.com/path?param1=value1&utm_source=wechat&campaign=promo123
```

### 3. Analytics Query

**Request**

```http
GET /api/v1/analytics/AbCd
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "short_code": "AbCd",
    "pv": 10000,
    "uv": 3500,
    "top_sources": [
      {"source": "wechat", "count": 4000},
      {"source": "weibo", "count": 2500},
      {"source": "direct", "count": 3500}
    ]
  }
}
```

## 7. Data Models

### Short Links Table (short_links)

| Field | Type | Description |
|-------|------|-------------|
| id | bigint | Primary key |
| short_code | varchar(6) | Short code |
| original_url | varchar(2048) | Original URL |
| params | json | Parameter template |
| created_at | datetime | Creation time |
| expire_at | datetime | Expiration time |
| status | tinyint | Status: 1-active, 0-disabled |

### Access Logs Table (access_logs)

| Field | Type | Description |
|-------|------|-------------|
| id | bigint | Primary key |
| short_code | varchar(6) | Short code |
| client_ip | varchar(64) | Client IP |
| user_agent | varchar(512) | User-Agent |
| referer | varchar(512) | Referrer page |
| access_time | datetime | Access time |

## 8. Core Features

### 1. Collision Detection

- Use Redis Bloom Filter for fast pre-check
- Perform exact database query on Bloom Filter false positives
- Increment hash value and re-encode on collision

### 2. PV/UV Analytics

- **PV (Page View)**: Count every visit, stored in Redis Counter
- **UV (Unique Visitor)**: Deduplicated statistics based on user Cookie/IP

### 3. Traffic Source Analysis

- Parse HTTP Referer header to get traffic source
- Count visits from each source
- Identify hot promotion channels

### 4. Async Log Processing

- Access logs are first written to RocketMQ
- Consumer batch processes and writes to data warehouse
- Ensures high availability, falls back to local logging on MQ failure
