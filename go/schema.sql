
-- 
-- 创建 posts 表
CREATE TABLE IF NOT EXISTS posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    file VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_file (file),
    INDEX idx_email (email)
);

