package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// 代理配置结构
type ProxyConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Cipher   string `yaml:"cipher"`
	UUID     string `yaml:"uuid"`
	AlterID  int    `yaml:"alterId"`
	Network  string `yaml:"network"`
	TLS      bool   `yaml:"tls"`
}

// Clash配置结构
type ClashConfig struct {
	Proxies []ProxyConfig `yaml:"proxies"`
}

// API请求结构
type ConvertRequest struct {
	ConfigSource string `json:"config_source"`
	ConfigURL    string `json:"config_url"`
	ConfigText   string `json:"config_text"`
}

// API响应结构
type ConvertResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	SubscriptionURL   string `json:"subscription_url,omitempty"`
	SubscriptionID    string `json:"subscription_id,omitempty"`
	ProxyCount        int    `json:"proxy_count,omitempty"`
	SubscriptionContent string `json:"subscription_content,omitempty"`
}

// 订阅配置结构
type SubscriptionConfig struct {
	ID              string    `json:"id"`
	ConfigHash      string    `json:"config_hash"`      // 配置哈希用于去重
	SourceURL       string    `json:"source_url,omitempty"`
	SourceContent   string    `json:"source_content,omitempty"`
	Content         string    `json:"content"`
	ProxyCount      int       `json:"proxy_count"`
	CreateTime      time.Time `json:"create_time"`
	LastUpdate      time.Time `json:"last_update"`
	IsAutoUpdate    bool      `json:"is_auto_update"`
}

// 管理员配置结构
type AdminConfig struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	IsSetup      bool   `json:"is_setup"`
}

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 全局变量存储多个订阅配置和管理员配置
var (
	db *sql.DB
	subscriptions = make(map[string]*SubscriptionConfig)      // subscriptionID -> config
	configHashMap = make(map[string]string)                   // configHash -> subscriptionID
	subscriptionsMux sync.RWMutex
	adminConfig = &AdminConfig{}
	adminMux sync.RWMutex
	activeSessions = make(map[string]time.Time) // sessionToken -> expireTime
	sessionsMux sync.RWMutex
)

// 将SS配置转换为URI
func ssToURI(proxy ProxyConfig) string {
	auth := fmt.Sprintf("%s:%s", proxy.Cipher, proxy.Password)
	authB64 := base64.StdEncoding.EncodeToString([]byte(auth))
	name := url.QueryEscape(proxy.Name)
	return fmt.Sprintf("ss://%s@%s:%d#%s", authB64, proxy.Server, proxy.Port, name)
}

// 将VMess配置转换为URI
func vmessToURI(proxy ProxyConfig) string {
	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   proxy.Name,
		"add":  proxy.Server,
		"port": strconv.Itoa(proxy.Port),
		"id":   proxy.UUID,
		"aid":  strconv.Itoa(proxy.AlterID),
		"net":  proxy.Network,
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
	}
	
	if proxy.TLS {
		vmessConfig["tls"] = "tls"
	}
	
	if proxy.Network == "" {
		vmessConfig["net"] = "tcp"
	}
	
	jsonBytes, _ := json.Marshal(vmessConfig)
	vmessB64 := base64.StdEncoding.EncodeToString(jsonBytes)
	return fmt.Sprintf("vmess://%s", vmessB64)
}

// 将Trojan配置转换为URI
func trojanToURI(proxy ProxyConfig) string {
	name := url.QueryEscape(proxy.Name)
	return fmt.Sprintf("trojan://%s@%s:%d#%s", proxy.Password, proxy.Server, proxy.Port, name)
}

// 转换Clash配置为订阅链接
func convertClashToSubscription(clashConfig ClashConfig) (string, int) {
	var subscriptionLines []string
	
	for _, proxy := range clashConfig.Proxies {
		var uri string
		switch proxy.Type {
		case "ss":
			uri = ssToURI(proxy)
		case "vmess":
			uri = vmessToURI(proxy)
		case "trojan":
			uri = trojanToURI(proxy)
		default:
			continue
		}
		subscriptionLines = append(subscriptionLines, uri)
	}
	
	content := strings.Join(subscriptionLines, "\n")
	subscriptionB64 := base64.StdEncoding.EncodeToString([]byte(content))
	
	return subscriptionB64, len(subscriptionLines)
}

// 从URL下载配置文件
func downloadConfigFromURL(configURL string) (string, error) {
	// 创建HTTP客户端，跳过SSL验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return string(body), nil
}

// 生成随机订阅ID
func generateSubscriptionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// 生成随机会话token
func generateSessionToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// 密码哈希
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// 生成配置哈希用于去重
func generateConfigHash(configSource, configURL, configText string) string {
	var data string
	if configSource == "url" {
		data = "url:" + configURL
	} else {
		// 对文本内容进行标准化处理，去除空白字符差异
		lines := strings.Split(strings.TrimSpace(configText), "\n")
		var cleanLines []string
		for _, line := range lines {
			cleanLine := strings.TrimSpace(line)
			if cleanLine != "" {
				cleanLines = append(cleanLines, cleanLine)
			}
		}
		sort.Strings(cleanLines) // 排序确保一致性
		data = "text:" + strings.Join(cleanLines, "\n")
	}
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// 查找已存在的配置
func findExistingConfig(configHash string) *SubscriptionConfig {
	subscriptionsMux.RLock()
	defer subscriptionsMux.RUnlock()
	
	if subscriptionID, exists := configHashMap[configHash]; exists {
		if config, exists := subscriptions[subscriptionID]; exists {
			return config
		}
	}
	return nil
}

// 初始化数据库
func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "subscription.db")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %v", err)
	}

	// 测试数据库连接
	if err = db.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 创建表结构
	if err = createTables(); err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	log.Println("数据库初始化完成")
	return nil
}

// 创建数据库表
func createTables() error {
	// 管理员配置表
	createAdminTable := `
	CREATE TABLE IF NOT EXISTS admin_config (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		is_setup BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 订阅配置表
	createSubscriptionTable := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		config_hash TEXT NOT NULL,
		source_url TEXT,
		source_content TEXT,
		content TEXT NOT NULL,
		proxy_count INTEGER DEFAULT 0,
		is_auto_update BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 会话管理表
	createSessionTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);`

	// 配置哈希映射表
	createHashMapTable := `
	CREATE TABLE IF NOT EXISTS config_hash_map (
		config_hash TEXT PRIMARY KEY,
		subscription_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (subscription_id) REFERENCES subscriptions(id)
	);`

	// 执行创建表的SQL
	tables := []string{createAdminTable, createSubscriptionTable, createSessionTable, createHashMapTable}
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_subscriptions_config_hash ON subscriptions(config_hash);",
		"CREATE INDEX IF NOT EXISTS idx_subscriptions_updated_at ON subscriptions(updated_at);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);",
		"CREATE INDEX IF NOT EXISTS idx_config_hash_map_subscription_id ON config_hash_map(subscription_id);",
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return err
		}
	}

	return nil
}

// 加载管理员配置
func loadAdminConfig() {
	adminMux.Lock()
	defer adminMux.Unlock()
	
	row := db.QueryRow("SELECT username, password_hash, is_setup FROM admin_config WHERE id = 1")
	err := row.Scan(&adminConfig.Username, &adminConfig.PasswordHash, &adminConfig.IsSetup)
	if err != nil {
		if err == sql.ErrNoRows {
			// 配置不存在，需要首次设置
			adminConfig.IsSetup = false
			return
		}
		log.Printf("加载管理员配置失败: %v", err)
		adminConfig.IsSetup = false
	}
}

// 保存管理员配置
func saveAdminConfig() error {
	adminMux.RLock()
	username := adminConfig.Username
	passwordHash := adminConfig.PasswordHash
	isSetup := adminConfig.IsSetup
	adminMux.RUnlock()
	
	// 使用 INSERT OR REPLACE 来插入或更新记录
	_, err := db.Exec(`
		INSERT OR REPLACE INTO admin_config (id, username, password_hash, is_setup, updated_at) 
		VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)`,
		username, passwordHash, isSetup)
	
	return err
}

// 验证会话token
func validateSession(token string) bool {
	if token == "" {
		return false
	}
	
	var expiresAt time.Time
	err := db.QueryRow("SELECT expires_at FROM sessions WHERE token = ?", token).Scan(&expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Printf("验证会话失败: %v", err)
		return false
	}
	
	if time.Now().After(expiresAt) {
		// 清理过期会话
		db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return false
	}
	
	return true
}

// 创建会话
func createSession() string {
	token := generateSessionToken()
	expireTime := time.Now().Add(24 * time.Hour) // 24小时有效期
	
	// 插入会话到数据库
	_, err := db.Exec("INSERT INTO sessions (token, expires_at) VALUES (?, ?)", token, expireTime)
	if err != nil {
		log.Printf("创建会话失败: %v", err)
		return ""
	}
	
	return token
}

// 清理过期会话
func cleanupExpiredSessions() {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	if err != nil {
		log.Printf("清理过期会话失败: %v", err)
	}
}

// 保存订阅配置到数据库
func saveSubscriptionToDB(config *SubscriptionConfig) error {
	subscriptionsMux.Lock()
	defer subscriptionsMux.Unlock()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()
	
	// 插入或更新订阅
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO subscriptions 
		(id, config_hash, source_url, source_content, content, proxy_count, is_auto_update, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		config.ID, config.ConfigHash, config.SourceURL, config.SourceContent, 
		config.Content, config.ProxyCount, config.IsAutoUpdate)
	if err != nil {
		return fmt.Errorf("保存订阅失败: %v", err)
	}
	
	// 更新配置哈希映射
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO config_hash_map (config_hash, subscription_id) 
		VALUES (?, ?)`, config.ConfigHash, config.ID)
	if err != nil {
		return fmt.Errorf("保存配置哈希映射失败: %v", err)
	}
	
	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}
	
	// 更新内存缓存
	subscriptions[config.ID] = config
	configHashMap[config.ConfigHash] = config.ID
	
	return nil
}

// 从数据库加载订阅配置
func loadSubscriptionFromDB(subscriptionID string) (*SubscriptionConfig, error) {
	config := &SubscriptionConfig{}
	var createdAt, updatedAt string
	
	row := db.QueryRow(`
		SELECT id, config_hash, source_url, source_content, content, proxy_count, 
		       is_auto_update, created_at, updated_at
		FROM subscriptions WHERE id = ?`, subscriptionID)
	
	err := row.Scan(&config.ID, &config.ConfigHash, &config.SourceURL, 
		&config.SourceContent, &config.Content, &config.ProxyCount, 
		&config.IsAutoUpdate, &createdAt, &updatedAt)
	
	if err != nil {
		return nil, err
	}
	
	// 解析时间
	if config.CreateTime, err = time.Parse("2006-01-02 15:04:05", createdAt); err != nil {
		config.CreateTime = time.Now()
	}
	if config.LastUpdate, err = time.Parse("2006-01-02 15:04:05", updatedAt); err != nil {
		config.LastUpdate = time.Now()
	}
	
	return config, nil
}

// 从数据库加载所有订阅配置
func loadAllSubscriptionsFromDB() error {
	subscriptionsMux.Lock()
	defer subscriptionsMux.Unlock()
	
	rows, err := db.Query(`
		SELECT id, config_hash, source_url, source_content, content, proxy_count, 
		       is_auto_update, created_at, updated_at
		FROM subscriptions`)
	if err != nil {
		return fmt.Errorf("查询订阅列表失败: %v", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		config := &SubscriptionConfig{}
		var createdAt, updatedAt string
		
		err := rows.Scan(&config.ID, &config.ConfigHash, &config.SourceURL, 
			&config.SourceContent, &config.Content, &config.ProxyCount, 
			&config.IsAutoUpdate, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("扫描订阅记录失败: %v", err)
			continue
		}
		
		// 解析时间
		if config.CreateTime, err = time.Parse("2006-01-02 15:04:05", createdAt); err != nil {
			config.CreateTime = time.Now()
		}
		if config.LastUpdate, err = time.Parse("2006-01-02 15:04:05", updatedAt); err != nil {
			config.LastUpdate = time.Now()
		}
		
		// 加载到内存缓存
		subscriptions[config.ID] = config
		configHashMap[config.ConfigHash] = config.ID
	}
	
	log.Printf("从数据库加载了 %d 个订阅配置", len(subscriptions))
	return nil
}

// 获取本机IP地址
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// 更新订阅内容
func updateSubscriptionContent(config *SubscriptionConfig) error {
	var configContent string
	var err error
	
	if config.SourceURL != "" {
		// 从URL下载最新配置
		configContent, err = downloadConfigFromURL(config.SourceURL)
		if err != nil {
			return fmt.Errorf("下载配置失败: %v", err)
		}
	} else {
		// 使用存储的内容
		configContent = config.SourceContent
	}
	
	// 解析YAML配置
	var clashConfig ClashConfig
	if err := yaml.Unmarshal([]byte(configContent), &clashConfig); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}
	
	// 转换为订阅链接
	subscriptionB64, proxyCount := convertClashToSubscription(clashConfig)
	
	// 更新配置
	config.Content = subscriptionB64
	config.ProxyCount = proxyCount
	config.LastUpdate = time.Now()
	
	// 保存到数据库
	if err := saveSubscriptionToDB(config); err != nil {
		return fmt.Errorf("保存更新到数据库失败: %v", err)
	}
	
	log.Printf("订阅 %s 更新成功，节点数量: %d", config.ID, proxyCount)
	return nil
}

// 检查并更新订阅内容（实时更新）
func checkAndUpdateSubscription(config *SubscriptionConfig) {
	// 只有URL来源的配置才需要更新
	if !config.IsAutoUpdate || config.SourceURL == "" {
		return
	}
	
	// 每次访问都实时更新，异步执行避免阻塞请求
	go func() {
		if err := updateSubscriptionContent(config); err != nil {
			log.Printf("更新订阅 %s 失败: %v", config.ID, err)
		} else {
			log.Printf("订阅 %s 已实时更新，节点数量: %d", config.ID, config.ProxyCount)
		}
	}()
}

// 首页处理器
func indexHandler(w http.ResponseWriter, r *http.Request) {
	// 检查管理员是否已设置
	adminMux.RLock()
	isSetup := adminConfig.IsSetup
	adminMux.RUnlock()
	
	if !isSetup {
		// 重定向到首次设置页面
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	
	tmpl := getIndexTemplate()
	
	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, nil)
}

// API转换处理器
func convertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ConvertResponse{
			Success: false,
			Message: "请求格式错误",
		}
		sendJSONResponse(w, response)
		return
	}
	
	var configContent string
	var err error
	
	switch req.ConfigSource {
	case "url":
		if req.ConfigURL == "" {
			response := ConvertResponse{
				Success: false,
				Message: "请输入配置文件URL",
			}
			sendJSONResponse(w, response)
			return
		}
		configContent, err = downloadConfigFromURL(req.ConfigURL)
		if err != nil {
			response := ConvertResponse{
				Success: false,
				Message: fmt.Sprintf("下载配置文件失败: %v", err),
			}
			sendJSONResponse(w, response)
			return
		}
	case "text":
		if req.ConfigText == "" {
			response := ConvertResponse{
				Success: false,
				Message: "请输入配置文件内容",
			}
			sendJSONResponse(w, response)
			return
		}
		configContent = req.ConfigText
	default:
		response := ConvertResponse{
			Success: false,
			Message: "无效的配置源类型",
		}
		sendJSONResponse(w, response)
		return
	}
	
	// 解析YAML配置
	var clashConfig ClashConfig
	if err := yaml.Unmarshal([]byte(configContent), &clashConfig); err != nil {
		response := ConvertResponse{
			Success: false,
			Message: fmt.Sprintf("配置文件格式错误: %v", err),
		}
		sendJSONResponse(w, response)
		return
	}
	
	// 生成配置哈希用于去重检查
	configHash := generateConfigHash(req.ConfigSource, req.ConfigURL, req.ConfigText)
	
	// 检查是否已存在相同配置
	if existingConfig := findExistingConfig(configHash); existingConfig != nil {
		// 如果是URL配置，更新一下内容以确保是最新的
		if existingConfig.IsAutoUpdate {
			go func() {
				if err := updateSubscriptionContent(existingConfig); err != nil {
					log.Printf("更新已存在订阅 %s 失败: %v", existingConfig.ID, err)
				}
			}()
		}
		
		// 生成订阅链接
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		subscriptionURL := fmt.Sprintf("%s://%s/subscription/%s", scheme, r.Host, existingConfig.ID)
		
		response := ConvertResponse{
			Success:           true,
			Message:           fmt.Sprintf("找到已存在的配置！节点数量: %d，订阅ID: %s", existingConfig.ProxyCount, existingConfig.ID),
			SubscriptionURL:   subscriptionURL,
			SubscriptionID:    existingConfig.ID,
			ProxyCount:        existingConfig.ProxyCount,
			SubscriptionContent: func() string {
				if len(existingConfig.Content) > 200 {
					return existingConfig.Content[:200] + "..."
				}
				return existingConfig.Content
			}(),
		}
		
		sendJSONResponse(w, response)
		return
	}
	
	// 转换为订阅链接
	subscriptionB64, proxyCount := convertClashToSubscription(clashConfig)
	
	if proxyCount == 0 {
		response := ConvertResponse{
			Success: false,
			Message: "未找到任何有效的代理配置",
		}
		sendJSONResponse(w, response)
		return
	}
	
	// 生成随机订阅ID
	subscriptionID := generateSubscriptionID()
	
	// 创建订阅配置
	now := time.Now()
	config := &SubscriptionConfig{
		ID:           subscriptionID,
		ConfigHash:   configHash,
		Content:      subscriptionB64,
		ProxyCount:   proxyCount,
		CreateTime:   now,
		LastUpdate:   now,
		IsAutoUpdate: req.ConfigSource == "url", // 只有URL来源才自动更新
	}
	
	if req.ConfigSource == "url" {
		config.SourceURL = req.ConfigURL
	} else {
		config.SourceContent = req.ConfigText
	}
	
	// 保存订阅配置到数据库
	if err := saveSubscriptionToDB(config); err != nil {
		log.Printf("保存订阅配置到数据库失败: %v", err)
		response := ConvertResponse{
			Success: false,
			Message: fmt.Sprintf("保存订阅配置失败: %v", err),
		}
		sendJSONResponse(w, response)
		return
	}
	
	// 生成订阅链接
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	subscriptionURL := fmt.Sprintf("%s://%s/subscription/%s", scheme, r.Host, subscriptionID)
	
	response := ConvertResponse{
		Success:           true,
		Message:           fmt.Sprintf("转换成功！找到 %d 个代理节点，订阅ID: %s", proxyCount, subscriptionID),
		SubscriptionURL:   subscriptionURL,
		SubscriptionID:    subscriptionID,
		ProxyCount:        proxyCount,
		SubscriptionContent: func() string {
			if len(subscriptionB64) > 200 {
				return subscriptionB64[:200] + "..."
			}
			return subscriptionB64
		}(),
	}
	
	sendJSONResponse(w, response)
}

// 订阅链接处理器
func subscriptionHandler(w http.ResponseWriter, r *http.Request) {
	// 解析URL路径，获取订阅ID
	path := strings.TrimPrefix(r.URL.Path, "/subscription")
	if path == "" || path == "/" {
		// 兼容旧版本，返回第一个订阅
		subscriptionsMux.RLock()
		for _, config := range subscriptions {
			subscriptionsMux.RUnlock()
			
			// 检查并更新订阅内容
			checkAndUpdateSubscription(config)
			
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=\"subscription.txt\"")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write([]byte(config.Content))
			return
		}
		subscriptionsMux.RUnlock()
		http.Error(w, "订阅文件不存在，请先生成订阅", http.StatusNotFound)
		return
	}
	
	// 提取订阅ID
	subscriptionID := strings.TrimPrefix(path, "/")
	
	// 查找订阅配置
	subscriptionsMux.RLock()
	config, exists := subscriptions[subscriptionID]
	subscriptionsMux.RUnlock()
	
	if !exists {
		// 尝试从数据库加载
		config, err := loadSubscriptionFromDB(subscriptionID)
		if err != nil {
			http.Error(w, "订阅ID不存在", http.StatusNotFound)
			return
		}
		
		// 检查并更新订阅内容
		checkAndUpdateSubscription(config)
		
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"subscription.txt\"")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Last-Modified", config.LastUpdate.Format(time.RFC1123))
		w.Write([]byte(config.Content))
		return
	}
	
	// 检查并更新订阅内容（实时更新）
	checkAndUpdateSubscription(config)
	
	// 返回订阅内容
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"subscription.txt\"")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Last-Modified", config.LastUpdate.Format(time.RFC1123))
	w.Write([]byte(config.Content))
}

// 首次设置处理器
func setupHandler(w http.ResponseWriter, r *http.Request) {
	adminMux.RLock()
	isSetup := adminConfig.IsSetup
	adminMux.RUnlock()
	
	if isSetup {
		// 已设置，重定向到首页
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	if r.Method == "GET" {
		// 显示设置页面
		tmpl, err := template.New("setup").Parse(setupTemplate)
		if err != nil {
			http.Error(w, "模板解析错误", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}
	
	if r.Method == "POST" {
		// 处理设置表单
		username := r.FormValue("username")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		
		if username == "" || password == "" {
			http.Error(w, "用户名和密码不能为空", http.StatusBadRequest)
			return
		}
		
		if password != confirmPassword {
			http.Error(w, "两次输入的密码不一致", http.StatusBadRequest)
			return
		}
		
		if len(password) < 6 {
			http.Error(w, "密码长度至少6位", http.StatusBadRequest)
			return
		}
		
		// 保存管理员配置
		adminMux.Lock()
		adminConfig.Username = username
		adminConfig.PasswordHash = hashPassword(password)
		adminConfig.IsSetup = true
		adminMux.Unlock()
		
		if err := saveAdminConfig(); err != nil {
			log.Printf("保存管理员配置失败: %v", err)
			http.Error(w, "保存配置失败", http.StatusInternalServerError)
			return
		}
		
		log.Printf("管理员账户设置完成，用户名: %s", username)
		
		// 重定向到首页
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
}

// 登录处理器
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.New("login").Parse(loginTemplate)
		if err != nil {
			http.Error(w, "模板解析错误", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}
	
	if r.Method == "POST" {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "请求格式错误", http.StatusBadRequest)
			return
		}
		
		adminMux.RLock()
		storedUsername := adminConfig.Username
		storedPasswordHash := adminConfig.PasswordHash
		adminMux.RUnlock()
		
		if req.Username != storedUsername || hashPassword(req.Password) != storedPasswordHash {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "用户名或密码错误",
			})
			return
		}
		
		// 创建会话
		token := createSession()
		
		// 设置Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "admin_session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   24 * 3600, // 24小时
		})
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "登录成功",
		})
	}
}

// 管理后台处理器
func adminHandler(w http.ResponseWriter, r *http.Request) {
	// 验证会话
	cookie, err := r.Cookie("admin_session")
	if err != nil || !validateSession(cookie.Value) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	
	tmpl, err := template.New("admin").Parse(adminTemplate)
	if err != nil {
		http.Error(w, "模板解析错误", http.StatusInternalServerError)
		return
	}
	
	tmpl.Execute(w, nil)
}

// 获取订阅列表API
func subscriptionListHandler(w http.ResponseWriter, r *http.Request) {
	// 验证会话
	cookie, err := r.Cookie("admin_session")
	if err != nil || !validateSession(cookie.Value) {
		http.Error(w, "未授权", http.StatusUnauthorized)
		return
	}
	
	subscriptionsMux.RLock()
	subs := make([]*SubscriptionConfig, 0, len(subscriptions))
	for _, config := range subscriptions {
		// 创建副本避免并发问题
		sub := &SubscriptionConfig{
			ID:           config.ID,
			SourceURL:    config.SourceURL,
			ProxyCount:   config.ProxyCount,
			CreateTime:   config.CreateTime,
			LastUpdate:   config.LastUpdate,
			IsAutoUpdate: config.IsAutoUpdate,
		}
		subs = append(subs, sub)
	}
	subscriptionsMux.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"subscriptions": subs,
	})
}

// 退出登录处理器
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("admin_session")
	if err == nil {
		// 删除会话
		sessionsMux.Lock()
		delete(activeSessions, cookie.Value)
		sessionsMux.Unlock()
	}
	
	// 清除Cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "admin_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	
	http.Redirect(w, r, "/", http.StatusFound)
}

// 发送JSON响应
func sendJSONResponse(w http.ResponseWriter, response ConvertResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

// 检测是否运行在服务模式
func isServiceMode() bool {
	// 检查是否由systemd启动
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}
	// 检查父进程是否为systemd
	ppid := os.Getppid()
	if ppid == 1 {
		return true
	}
	return false
}

// 等待停止信号
func waitForShutdown() {
	if isServiceMode() {
		// 服务模式：等待系统信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("收到停止信号，正在关闭服务器...")
	} else {
		// 交互模式：等待用户输入
		fmt.Println("\n按回车键停止服务器...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		fmt.Println("👋 服务器已停止，再见！")
	}
}

func main() {
	// 初始化数据库
	if err := initDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()
	
	// 加载管理员配置
	loadAdminConfig()
	
	// 加载所有订阅配置
	if err := loadAllSubscriptionsFromDB(); err != nil {
		log.Printf("加载订阅配置失败: %v", err)
	}
	
	// 启动会话清理器
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredSessions()
		}
	}()
	
	// 设置路由
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/setup", setupHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/api/convert", convertHandler)
	http.HandleFunc("/api/subscriptions", subscriptionListHandler)
	http.HandleFunc("/subscription", subscriptionHandler)
	http.HandleFunc("/subscription/", subscriptionHandler) // 支持订阅ID路径
	
	// 获取本机IP
	localIP := getLocalIP()
	port := "8856"
	
	// 启动信息
	fmt.Println("🚀 订阅转换服务器 (Go版) 启动中...")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("📡 本地访问: http://localhost:%s\n", port)
	fmt.Printf("🌐 局域网访问: http://%s:%s\n", localIP, port)
	fmt.Printf("📋 订阅链接: http://%s:%s/subscription\n", localIP, port)
	fmt.Println("")
	fmt.Println("📋 使用说明:")
	fmt.Println("1. 在浏览器中打开上述地址")
	fmt.Println("2. 输入Clash配置文件URL或直接粘贴配置内容")
	fmt.Println("3. 点击'生成订阅链接'按钮")
	fmt.Println("4. 复制生成的订阅链接到你的代理客户端")
	fmt.Println("")
	fmt.Println("🛑 按回车键停止服务器")
	fmt.Println(strings.Repeat("=", 50))
	
	// 启动服务器
	server := &http.Server{
		Addr:    ":" + port,
		Handler: nil,
	}
	
	// 在goroutine中启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()
	
	// 等待停止信号或用户输入
	waitForShutdown()
} 