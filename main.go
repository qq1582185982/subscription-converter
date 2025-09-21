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

	_ "github.com/glebarez/go-sqlite"
	"gopkg.in/yaml.v3"
)

// 代理配置结构
type ProxyConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password,omitempty"`
	Cipher   string `yaml:"cipher"`
	UUID     string `yaml:"uuid,omitempty"`
	AlterID  int    `yaml:"alterId"`
	Network  string `yaml:"network,omitempty"`
	TLS      bool   `yaml:"tls,omitempty"`
	Security string `yaml:"security,omitempty"`
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

// 反向转换请求结构（订阅转Clash）
type ToClashRequest struct {
	ConfigSource string `json:"config_source"`
	ConfigURL    string `json:"config_url"`
	ConfigText   string `json:"config_text"`
}

// 反向转换响应结构
type ToClashResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ClashURL    string `json:"clash_url,omitempty"`
	ClashID     string `json:"clash_id,omitempty"`
	ProxyCount  int    `json:"proxy_count,omitempty"`
}

// Clash配置存储结构
type ClashConfigData struct {
	ID           string    `json:"id"`
	ConfigHash   string    `json:"config_hash"`   // 配置哈希用于去重
	SourceURL    string    `json:"source_url,omitempty"`
	SourceContent string   `json:"source_content,omitempty"`
	ClashConfig  string    `json:"clash_config"`  // 完整的Clash YAML配置
	ProxyCount   int       `json:"proxy_count"`
	CreateTime   time.Time `json:"create_time"`
	LastUpdate   time.Time `json:"last_update"`
	IsAutoUpdate bool      `json:"is_auto_update"`
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
	clashConfigs = make(map[string]*ClashConfigData)          // clashID -> config
	clashConfigHashMap = make(map[string]string)              // configHash -> clashID
	clashConfigsMux sync.RWMutex
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

// 解析Clash YAML配置，提取代理列表
func parseClashYAML(content string) ([]ProxyConfig, error) {
	log.Printf("开始解析Clash YAML配置，内容长度: %d", len(content))

	var clashConfig ClashConfig
	if err := yaml.Unmarshal([]byte(content), &clashConfig); err != nil {
		return nil, fmt.Errorf("解析Clash YAML失败: %v", err)
	}

	log.Printf("从Clash配置中提取到 %d 个代理节点", len(clashConfig.Proxies))
	return clashConfig.Proxies, nil
}

// 解析Base64编码的订阅内容
func parseSubscriptionContent(content string) ([]ProxyConfig, error) {
	// 尝试Base64解码
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		// 如果不是Base64编码，直接使用原内容
		decoded = []byte(content)
	}

	lines := strings.Split(string(decoded), "\n")
	var proxies []ProxyConfig

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var proxy ProxyConfig
		var err error

		if strings.HasPrefix(line, "ss://") {
			proxy, err = parseSSURI(line)
		} else if strings.HasPrefix(line, "vmess://") {
			proxy, err = parseVMessURI(line)
		} else if strings.HasPrefix(line, "trojan://") {
			proxy, err = parseTrojanURI(line)
		} else {
			log.Printf("跳过不支持的协议: %s", line[:min(50, len(line))])
			continue
		}

		if err != nil {
			log.Printf("解析URI失败: %v, URI: %s", err, line[:min(100, len(line))])
			continue
		}

		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

// 解析SS URI格式 ss://method:password@server:port#name
func parseSSURI(uri string) (ProxyConfig, error) {
	var proxy ProxyConfig

	// 移除 ss:// 前缀
	uri = strings.TrimPrefix(uri, "ss://")

	// 检查是否有fragment（节点名称）
	parts := strings.Split(uri, "#")
	if len(parts) == 2 {
		name, _ := url.QueryUnescape(parts[1])
		proxy.Name = name
		uri = parts[0]
	}

	// 尝试解析Base64编码的部分
	if !strings.Contains(uri, "@") {
		// 添加必要的padding
		switch len(uri) % 4 {
		case 2:
			uri += "=="
		case 3:
			uri += "="
		}

		decoded, err := base64.StdEncoding.DecodeString(uri)
		if err != nil {
			// 尝试URL安全的Base64解码
			decoded, err = base64.URLEncoding.DecodeString(uri)
			if err != nil {
				return proxy, fmt.Errorf("无效的SS Base64编码: %v", err)
			}
		}
		uri = string(decoded)
	}

	// 解析 method:password@server:port
	atIndex := strings.LastIndex(uri, "@")
	if atIndex == -1 {
		return proxy, fmt.Errorf("无效的SS URI格式")
	}

	// 解析认证部分
	auth := uri[:atIndex]
	colonIndex := strings.Index(auth, ":")
	if colonIndex == -1 {
		return proxy, fmt.Errorf("无效的SS认证格式")
	}

	proxy.Type = "ss"
	proxy.Cipher = auth[:colonIndex]
	proxy.Password = auth[colonIndex+1:]

	// 解析服务器地址部分
	serverPart := uri[atIndex+1:]
	lastColonIndex := strings.LastIndex(serverPart, ":")
	if lastColonIndex == -1 {
		return proxy, fmt.Errorf("无效的SS服务器格式")
	}

	proxy.Server = serverPart[:lastColonIndex]
	port, err := strconv.Atoi(serverPart[lastColonIndex+1:])
	if err != nil {
		return proxy, fmt.Errorf("无效的端口号: %v", err)
	}
	proxy.Port = port

	if proxy.Name == "" {
		proxy.Name = fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)
	}

	return proxy, nil
}

// 解析VMess URI格式
func parseVMessURI(uri string) (ProxyConfig, error) {
	var proxy ProxyConfig

	// 移除 vmess:// 前缀
	uri = strings.TrimPrefix(uri, "vmess://")

	// 添加必要的padding
	switch len(uri) % 4 {
	case 2:
		uri += "=="
	case 3:
		uri += "="
	}

	// Base64解码
	decoded, err := base64.StdEncoding.DecodeString(uri)
	if err != nil {
		// 尝试URL安全的Base64解码
		decoded, err = base64.URLEncoding.DecodeString(uri)
		if err != nil {
			return proxy, fmt.Errorf("无效的VMess Base64编码: %v", err)
		}
	}

	// 解析JSON
	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
		return proxy, fmt.Errorf("无效的VMess JSON格式: %v", err)
	}

	proxy.Type = "vmess"
	proxy.Name = getString(vmessConfig, "ps")
	proxy.Server = getString(vmessConfig, "add")
	proxy.Port = getInt(vmessConfig, "port")
	proxy.UUID = getString(vmessConfig, "id")
	proxy.AlterID = getInt(vmessConfig, "aid")
	proxy.Network = getString(vmessConfig, "net")
	proxy.TLS = getString(vmessConfig, "tls") == "tls"
	proxy.Cipher = "auto" // VMess默认cipher值

	// 设置security字段，默认为none
	security := getString(vmessConfig, "scy")
	if security == "" {
		security = "none"
	}
	proxy.Security = security

	if proxy.Name == "" {
		proxy.Name = fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)
	}

	return proxy, nil
}

// 解析Trojan URI格式 trojan://password@server:port#name
func parseTrojanURI(uri string) (ProxyConfig, error) {
	var proxy ProxyConfig

	// 移除 trojan:// 前缀
	uri = strings.TrimPrefix(uri, "trojan://")

	// 检查是否有fragment（节点名称）
	parts := strings.Split(uri, "#")
	if len(parts) == 2 {
		name, _ := url.QueryUnescape(parts[1])
		proxy.Name = name
		uri = parts[0]
	}

	// 解析 password@server:port
	atIndex := strings.LastIndex(uri, "@")
	if atIndex == -1 {
		return proxy, fmt.Errorf("无效的Trojan URI格式")
	}

	proxy.Type = "trojan"
	proxy.Password = uri[:atIndex]

	// 解析服务器地址部分
	serverPart := uri[atIndex+1:]
	lastColonIndex := strings.LastIndex(serverPart, ":")
	if lastColonIndex == -1 {
		return proxy, fmt.Errorf("无效的Trojan服务器格式")
	}

	proxy.Server = serverPart[:lastColonIndex]
	port, err := strconv.Atoi(serverPart[lastColonIndex+1:])
	if err != nil {
		return proxy, fmt.Errorf("无效的端口号: %v", err)
	}
	proxy.Port = port

	if proxy.Name == "" {
		proxy.Name = fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)
	}

	return proxy, nil
}

// 辅助函数：从map中安全获取字符串
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：从map中安全获取整数
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

// 辅助函数：获取两个数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 转换Clash配置为订阅链接
func convertClashToSubscription(clashConfig ClashConfig) (string, int) {
	var subscriptionLines []string
	
	log.Printf("开始转换 %d 个代理节点", len(clashConfig.Proxies))
	
	for i, proxy := range clashConfig.Proxies {
		var uri string
		log.Printf("处理节点 %d: 类型=%s, 名称=%s", i+1, proxy.Type, proxy.Name)
		
		switch proxy.Type {
		case "ss":
			uri = ssToURI(proxy)
			log.Printf("生成SS URI: %s", func() string {
				if len(uri) > 100 {
					return uri[:100] + "..."
				}
				return uri
			}())
		case "vmess":
			uri = vmessToURI(proxy)
			log.Printf("生成VMess URI: %s", func() string {
				if len(uri) > 100 {
					return uri[:100] + "..."
				}
				return uri
			}())
		case "trojan":
			uri = trojanToURI(proxy)
			log.Printf("生成Trojan URI: %s", func() string {
				if len(uri) > 100 {
					return uri[:100] + "..."
				}
				return uri
			}())
		default:
			log.Printf("跳过不支持的节点类型: %s", proxy.Type)
			continue
		}
		
		if uri != "" {
			subscriptionLines = append(subscriptionLines, uri)
		}
	}
	
	log.Printf("成功转换 %d 个节点", len(subscriptionLines))
	
	content := strings.Join(subscriptionLines, "\n")
	subscriptionB64 := base64.StdEncoding.EncodeToString([]byte(content))
	
	return subscriptionB64, len(subscriptionLines)
}

// 从URL下载配置文件，支持订阅链接和Clash配置
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

	content := string(body)

	// 检测内容类型并处理
	contentType := detectContentType(content)
	log.Printf("检测到内容类型: %s", contentType)

	switch contentType {
	case "subscription":
		// 如果是订阅链接内容，转换为Clash配置格式
		return convertSubscriptionToClash(content)
	case "clash":
		// 如果是Clash配置，直接返回
		return content, nil
	default:
		// 未知格式，尝试作为订阅处理
		log.Printf("未知内容格式，尝试作为订阅处理")
		converted, err := convertSubscriptionToClash(content)
		if err != nil {
			// 如果订阅解析失败，返回原内容（可能是其他格式的Clash配置）
			return content, nil
		}
		return converted, nil
	}
}

// 检测内容类型
func detectContentType(content string) string {
	content = strings.TrimSpace(content)

	// 首先检查是否为YAML格式的Clash配置
	if strings.Contains(content, "proxies:") ||
	   strings.Contains(content, "proxy-groups:") ||
	   strings.Contains(content, "rules:") {
		return "clash"
	}

	// 检查是否包含URI格式的代理
	if strings.Contains(content, "ss://") ||
	   strings.Contains(content, "vmess://") ||
	   strings.Contains(content, "trojan://") {
		return "subscription"
	}

	// 最后检查是否为Base64编码的订阅
	if isBase64Subscription(content) {
		return "subscription"
	}

	return "unknown"
}

// 检查是否为Base64编码的订阅
func isBase64Subscription(content string) bool {
	// 移除换行符和空格
	cleaned := strings.ReplaceAll(strings.ReplaceAll(content, "\n", ""), " ", "")

	// 检查是否为有效的Base64编码
	if _, err := base64.StdEncoding.DecodeString(cleaned); err != nil {
		return false
	}

	// 解码并检查内容
	decoded, _ := base64.StdEncoding.DecodeString(cleaned)
	decodedStr := string(decoded)

	// 检查解码后的内容是否包含代理URI
	return strings.Contains(decodedStr, "://")
}

// 将订阅内容转换为Clash配置格式
func convertSubscriptionToClash(content string) (string, error) {
	log.Printf("开始解析订阅内容，内容长度: %d", len(content))

	// 检测内容类型
	contentType := detectContentType(content)
	log.Printf("检测到内容类型: %s", contentType)

	var proxies []ProxyConfig
	var err error

	if contentType == "clash" {
		// 已经是Clash配置，直接解析YAML
		proxies, err = parseClashYAML(content)
		if err != nil {
			return "", fmt.Errorf("解析Clash YAML失败: %v", err)
		}
	} else {
		// 是订阅内容，需要解析URI
		proxies, err = parseSubscriptionContent(content)
		if err != nil {
			return "", fmt.Errorf("解析订阅内容失败: %v", err)
		}
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("未找到任何有效的代理配置")
	}

	log.Printf("成功解析 %d 个代理节点", len(proxies))

	// 构造Clash配置
	clashConfig := ClashConfig{
		Proxies: proxies,
	}

	// 将配置转换为YAML格式
	yamlData, err := yaml.Marshal(&clashConfig)
	if err != nil {
		return "", fmt.Errorf("生成Clash配置失败: %v", err)
	}

	return string(yamlData), nil
}

// 完整的Clash配置结构
type FullClashConfig struct {
	Port               int                    `yaml:"port"`
	SocksPort          int                    `yaml:"socks-port"`
	RedirPort          int                    `yaml:"redir-port,omitempty"`
	MixedPort          int                    `yaml:"mixed-port,omitempty"`
	AllowLan           bool                   `yaml:"allow-lan"`
	Mode               string                 `yaml:"mode"`
	LogLevel           string                 `yaml:"log-level"`
	ExternalController string                 `yaml:"external-controller"`
	DNS                map[string]interface{} `yaml:"dns"`
	Proxies            []ProxyConfig          `yaml:"proxies"`
	ProxyGroups        []ProxyGroup           `yaml:"proxy-groups"`
	Rules              []string               `yaml:"rules"`
}

// 代理组结构
type ProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
	URL     string   `yaml:"url,omitempty"`
	Interval int     `yaml:"interval,omitempty"`
}

// 生成完整的Clash配置（订阅转Clash）
func generateFullClashConfig(content string) (string, int, error) {
	log.Printf("开始生成完整Clash配置，内容长度: %d", len(content))

	// 检测内容类型
	contentType := detectContentType(content)
	log.Printf("检测到内容类型: %s", contentType)

	var proxies []ProxyConfig
	var err error

	if contentType == "clash" {
		// 已经是Clash配置，直接解析YAML
		proxies, err = parseClashYAML(content)
		if err != nil {
			return "", 0, fmt.Errorf("解析Clash YAML失败: %v", err)
		}
	} else {
		// 是订阅内容，需要解析URI
		proxies, err = parseSubscriptionContent(content)
		if err != nil {
			return "", 0, fmt.Errorf("解析订阅内容失败: %v", err)
		}
	}

	if len(proxies) == 0 {
		return "", 0, fmt.Errorf("未找到任何有效的代理配置")
	}

	log.Printf("成功解析 %d 个代理节点", len(proxies))

	// 生成代理名称列表
	var proxyNames []string
	for _, proxy := range proxies {
		proxyNames = append(proxyNames, proxy.Name)
	}

	// 构造完整的Clash配置
	fullConfig := FullClashConfig{
		Port:               7890,
		SocksPort:          7891,
		MixedPort:          7892,
		AllowLan:           false,
		Mode:               "Rule",
		LogLevel:           "info",
		ExternalController: "127.0.0.1:9090",
		DNS: map[string]interface{}{
			"enable":            true,
			"ipv6":              false,
			"default-nameserver": []string{"223.5.5.5", "119.29.29.29"},
			"enhanced-mode":     "fake-ip",
			"fake-ip-range":     "198.18.0.1/16",
			"nameserver": []string{
				"https://doh.pub/dns-query",
				"https://dns.alidns.com/dns-query",
			},
			"fallback": []string{
				"https://cloudflare-dns.com/dns-query",
				"https://dns.google/dns-query",
			},
		},
		Proxies: proxies,
		ProxyGroups: []ProxyGroup{
			{
				Name:     "🔰 节点选择",
				Type:     "select",
				Proxies:  append([]string{"♻️ 自动选择", "🎯 全球直连"}, proxyNames...),
			},
			{
				Name:     "♻️ 自动选择",
				Type:     "url-test",
				Proxies:  proxyNames,
				URL:      "http://www.gstatic.com/generate_204",
				Interval: 300,
			},
			{
				Name:    "🎯 全球直连",
				Type:    "select",
				Proxies: []string{"DIRECT"},
			},
			{
				Name:    "🛑 全球拦截",
				Type:    "select",
				Proxies: []string{"REJECT"},
			},
			{
				Name:    "🐟 漏网之鱼",
				Type:    "select",
				Proxies: []string{"🔰 节点选择", "🎯 全球直连"},
			},
		},
		Rules: []string{
			// 去广告规则
			"RULE-SET,reject,🛑 全球拦截",
			// 国内直连规则
			"RULE-SET,china,🎯 全球直连",
			"RULE-SET,cncidr,🎯 全球直连",
			// 国外代理规则
			"RULE-SET,proxy,🔰 节点选择",
			"RULE-SET,telegramcidr,🔰 节点选择",
			// 本地局域网直连
			"IP-CIDR,127.0.0.0/8,🎯 全球直连",
			"IP-CIDR,172.16.0.0/12,🎯 全球直连",
			"IP-CIDR,192.168.0.0/16,🎯 全球直连",
			"IP-CIDR,10.0.0.0/8,🎯 全球直连",
			// GeoIP 规则
			"GEOIP,CN,🎯 全球直连",
			// 漏网之鱼
			"MATCH,🐟 漏网之鱼",
		},
	}

	// 将配置转换为YAML格式
	yamlData, err := yaml.Marshal(&fullConfig)
	if err != nil {
		return "", 0, fmt.Errorf("生成Clash配置失败: %v", err)
	}

	return string(yamlData), len(proxies), nil
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
	db, err = sql.Open("sqlite", "subscription.db")
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

		// 检测文本内容类型并处理
		contentType := detectContentType(req.ConfigText)
		log.Printf("文本输入检测到内容类型: %s", contentType)

		if contentType == "subscription" {
			// 如果是订阅内容，转换为Clash配置
			converted, err := convertSubscriptionToClash(req.ConfigText)
			if err != nil {
				response := ConvertResponse{
					Success: false,
					Message: fmt.Sprintf("解析订阅内容失败: %v", err),
				}
				sendJSONResponse(w, response)
				return
			}
			configContent = converted
		} else {
			configContent = req.ConfigText
		}
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
		log.Printf("YAML解析失败: %v", err)
		log.Printf("配置内容前500字符: %s", func() string {
			if len(configContent) > 500 {
				return configContent[:500]
			}
			return configContent
		}())

		// 如果YAML解析失败，尝试作为订阅内容处理
		log.Printf("尝试将内容作为订阅链接处理")
		converted, subscriptionErr := convertSubscriptionToClash(configContent)
		if subscriptionErr != nil {
			// 订阅解析也失败，返回原始YAML错误
			response := ConvertResponse{
				Success: false,
				Message: fmt.Sprintf("配置文件格式错误 (YAML解析失败: %v, 订阅解析失败: %v)", err, subscriptionErr),
			}
			sendJSONResponse(w, response)
			return
		}

		// 订阅解析成功，使用转换后的Clash配置
		configContent = converted
		log.Printf("成功将订阅内容转换为Clash配置")

		// 重新解析转换后的YAML配置
		if err := yaml.Unmarshal([]byte(configContent), &clashConfig); err != nil {
			response := ConvertResponse{
				Success: false,
				Message: fmt.Sprintf("转换后的配置解析失败: %v", err),
			}
			sendJSONResponse(w, response)
			return
		}
	}
	
	log.Printf("成功解析YAML配置，找到 %d 个代理节点", len(clashConfig.Proxies))
	for i, proxy := range clashConfig.Proxies {
		log.Printf("节点 %d: 类型=%s, 名称=%s, 服务器=%s", i+1, proxy.Type, proxy.Name, proxy.Server)
		if i >= 5 { // 只显示前5个节点
			log.Printf("... 还有 %d 个节点", len(clashConfig.Proxies)-5)
			break
		}
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

// 反向转换处理器（订阅转Clash）
func toClashHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToClashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ToClashResponse{
			Success: false,
			Message: "请求格式错误",
		}
		sendToClashResponse(w, response)
		return
	}

	var configContent string
	var err error

	switch req.ConfigSource {
	case "url":
		if req.ConfigURL == "" {
			response := ToClashResponse{
				Success: false,
				Message: "请输入订阅链接URL",
			}
			sendToClashResponse(w, response)
			return
		}
		configContent, err = downloadConfigFromURL(req.ConfigURL)
		if err != nil {
			response := ToClashResponse{
				Success: false,
				Message: fmt.Sprintf("下载订阅内容失败: %v", err),
			}
			sendToClashResponse(w, response)
			return
		}
	case "text":
		if req.ConfigText == "" {
			response := ToClashResponse{
				Success: false,
				Message: "请输入订阅内容",
			}
			sendToClashResponse(w, response)
			return
		}
		configContent = req.ConfigText
	default:
		response := ToClashResponse{
			Success: false,
			Message: "无效的配置源类型",
		}
		sendToClashResponse(w, response)
		return
	}

	// 检测内容类型并处理
	contentType := detectContentType(configContent)
	log.Printf("反向转换检测到内容类型: %s", contentType)

	var clashConfig string
	var proxyCount int

	if contentType == "clash" {
		// 如果已经是Clash配置，直接使用
		log.Printf("内容已经是Clash配置，直接使用")
		clashConfig = configContent

		// 尝试解析节点数量
		var clashConfigStruct ClashConfig
		if yaml.Unmarshal([]byte(configContent), &clashConfigStruct) == nil {
			proxyCount = len(clashConfigStruct.Proxies)
			log.Printf("解析到 %d 个代理节点", proxyCount)
		} else {
			log.Printf("无法解析Clash配置中的节点数量")
		}
	} else {
		// 作为订阅内容处理，生成完整Clash配置
		log.Printf("作为订阅内容处理")
		var err error
		clashConfig, proxyCount, err = generateFullClashConfig(configContent)
		if err != nil {
			response := ToClashResponse{
				Success: false,
				Message: fmt.Sprintf("生成Clash配置失败: %v", err),
			}
			sendToClashResponse(w, response)
			return
		}
	}

	// 生成配置哈希用于去重检查
	configHash := generateConfigHash(req.ConfigSource, req.ConfigURL, req.ConfigText)

	// 检查是否已存在相同配置
	clashConfigsMux.RLock()
	if existingClashID, exists := clashConfigHashMap[configHash]; exists {
		if existingConfig, exists := clashConfigs[existingClashID]; exists {
			clashConfigsMux.RUnlock()

			// 如果是URL配置，更新一下内容以确保是最新的
			if existingConfig.IsAutoUpdate {
				go func() {
					if err := updateClashConfig(existingConfig); err != nil {
						log.Printf("更新已存在Clash配置 %s 失败: %v", existingConfig.ID, err)
					}
				}()
			}

			// 生成Clash配置链接
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			clashURL := fmt.Sprintf("%s://%s/clash-config/%s.yaml", scheme, r.Host, existingConfig.ID)

			response := ToClashResponse{
				Success: true,
				Message: fmt.Sprintf("找到已存在的配置！节点数量: %d，配置ID: %s", existingConfig.ProxyCount, existingConfig.ID),
				ClashURL: clashURL,
				ClashID:  existingConfig.ID,
				ProxyCount: existingConfig.ProxyCount,
			}

			sendToClashResponse(w, response)
			return
		}
	}
	clashConfigsMux.RUnlock()

	// 生成随机Clash配置ID
	clashID := generateSubscriptionID()

	// 创建Clash配置数据
	now := time.Now()
	config := &ClashConfigData{
		ID:           clashID,
		ConfigHash:   configHash,
		ClashConfig:  clashConfig,
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

	// 保存Clash配置到内存
	clashConfigsMux.Lock()
	clashConfigs[clashID] = config
	clashConfigHashMap[configHash] = clashID
	clashConfigsMux.Unlock()

	log.Printf("创建新Clash配置: ID=%s, 节点数量=%d", clashID, proxyCount)

	// 生成Clash配置链接
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	clashURL := fmt.Sprintf("%s://%s/clash-config/%s.yaml", scheme, r.Host, clashID)

	response := ToClashResponse{
		Success:    true,
		Message:    fmt.Sprintf("转换成功！生成包含 %d 个节点的Clash配置", proxyCount),
		ClashURL:   clashURL,
		ClashID:    clashID,
		ProxyCount: proxyCount,
	}

	sendToClashResponse(w, response)
}

// 发送反向转换响应
func sendToClashResponse(w http.ResponseWriter, response ToClashResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
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

// Clash配置文件处理器
func clashConfigHandler(w http.ResponseWriter, r *http.Request) {
	// 解析URL路径，获取配置ID
	path := strings.TrimPrefix(r.URL.Path, "/clash-config/")
	if path == "" || path == "/" {
		http.Error(w, "Clash配置ID不能为空", http.StatusBadRequest)
		return
	}

	// 提取文件名（移除.yaml后缀）
	clashID := strings.TrimSuffix(path, ".yaml")
	if clashID == path {
		// 如果没有.yaml后缀，也尝试处理
		clashID = path
	}

	log.Printf("请求Clash配置ID: %s", clashID)

	clashConfigsMux.RLock()
	config, exists := clashConfigs[clashID]
	clashConfigsMux.RUnlock()

	if !exists {
		log.Printf("Clash配置不存在: %s", clashID)
		http.Error(w, "Clash配置不存在", http.StatusNotFound)
		return
	}

	// 如果是自动更新的配置，检查是否需要更新
	if config.IsAutoUpdate && config.SourceURL != "" {
		go func() {
			if err := updateClashConfig(config); err != nil {
				log.Printf("更新Clash配置 %s 失败: %v", clashID, err)
			} else {
				log.Printf("Clash配置 %s 已实时更新，节点数量: %d", clashID, config.ProxyCount)
			}
		}()
	}

	// 设置响应头
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"clash-%s.yaml\"", clashID))
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Printf("返回Clash配置: %s，节点数量: %d", clashID, config.ProxyCount)

	// 返回配置内容
	w.Write([]byte(config.ClashConfig))
}

// 更新Clash配置内容
func updateClashConfig(config *ClashConfigData) error {
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

	// 检测内容类型
	contentType := detectContentType(configContent)
	log.Printf("更新Clash配置 %s，检测到内容类型: %s", config.ID, contentType)

	var clashConfig string
	var proxyCount int

	if contentType == "clash" {
		// 已经是Clash配置，直接使用
		clashConfig = configContent

		// 解析并计算节点数量
		var clashObj ClashConfig
		if err := yaml.Unmarshal([]byte(configContent), &clashObj); err == nil {
			proxyCount = len(clashObj.Proxies)
		}
		log.Printf("使用现有Clash配置，节点数量: %d", proxyCount)
	} else {
		// 是订阅内容，需要转换为Clash配置
		clashConfig, proxyCount, err = generateFullClashConfig(configContent)
		if err != nil {
			return fmt.Errorf("生成Clash配置失败: %v", err)
		}
		log.Printf("从订阅生成Clash配置，节点数量: %d", proxyCount)
	}

	// 更新配置
	clashConfigsMux.Lock()
	config.ClashConfig = clashConfig
	config.ProxyCount = proxyCount
	config.LastUpdate = time.Now()
	clashConfigsMux.Unlock()

	log.Printf("Clash配置 %s 更新成功，节点数量: %d", config.ID, proxyCount)
	return nil
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
	http.HandleFunc("/api/to-clash", toClashHandler)
	http.HandleFunc("/api/subscriptions", subscriptionListHandler)
	http.HandleFunc("/subscription", subscriptionHandler)
	http.HandleFunc("/subscription/", subscriptionHandler) // 支持订阅ID路径
	http.HandleFunc("/clash-config/", clashConfigHandler)   // 支持Clash配置访问
	
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