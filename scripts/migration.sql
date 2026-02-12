-- Short Link Service Database Schema

-- Create database
CREATE DATABASE IF NOT EXISTS shortlink CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE shortlink;

-- Short links table
CREATE TABLE IF NOT EXISTS short_links (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    short_code VARCHAR(6) UNIQUE NOT NULL COMMENT 'Short code for the link',
    original_url VARCHAR(2048) NOT NULL COMMENT 'Original long URL',
    params JSON COMMENT 'Additional parameters for the link',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation timestamp',
    expire_at DATETIME COMMENT 'Expiration timestamp (optional)',
    status TINYINT DEFAULT 1 COMMENT '1=active, 0=disabled',
    INDEX idx_short_code (short_code),
    INDEX idx_expire_at (expire_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Short links storage';

-- Access logs table
CREATE TABLE IF NOT EXISTS access_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    short_code VARCHAR(6) NOT NULL COMMENT 'Short code of the accessed link',
    client_ip VARCHAR(64) COMMENT 'Client IP address',
    user_agent VARCHAR(512) COMMENT 'User-Agent header',
    referer VARCHAR(512) COMMENT 'Referer header',
    access_time DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT 'Access timestamp',
    INDEX idx_short_code (short_code),
    INDEX idx_access_time (access_time),
    INDEX idx_client_ip (client_ip)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Access logs for analytics';
