package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ==================== 角色定义 ====================

const (
	RoleAdmin     = "admin"     // 系统管理员 - 完全控制
	RoleAnalyst   = "analyst"   // 安全分析师 - 查看/处理告警、报告
	RoleDeveloper = "developer" // DApp开发者 - 查看监控数据、告警
	RoleOperator  = "operator"  // 运维人员 - 系统状态、审计日志
	RoleUser      = "user"      // 普通用户 - 仪表板、风险事件查看
)

// AllRoles 所有有效角色
var AllRoles = []string{RoleAdmin, RoleAnalyst, RoleDeveloper, RoleOperator, RoleUser}

// RoleLabels 角色中文标签
var RoleLabels = map[string]string{
	RoleAdmin:     "系统管理员",
	RoleAnalyst:   "安全分析师",
	RoleDeveloper: "DApp开发者",
	RoleOperator:  "运维人员",
	RoleUser:      "普通用户",
}

// IsValidRole 验证角色是否合法
func IsValidRole(role string) bool {
	for _, r := range AllRoles {
		if r == role {
			return true
		}
	}
	return false
}

// ==================== 错误定义 ====================

var (
	ErrUserExists         = errors.New("用户名或邮箱已存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrForbidden          = errors.New("权限不足")
	ErrInvalidRole        = errors.New("无效的角色")
)

// ==================== 数据模型 ====================

// User 用户模型
type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // 不返回给前端
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// AuditLogEntry 审计日志条目
type AuditLogEntry struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource,omitempty"`
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ==================== 请求/响应 ====================

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"` // 可选，默认 user
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      *User  `json:"user"`
}

// UserListResult 用户分页列表
type UserListResult struct {
	Items    []*User `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	Pages    int     `json:"pages"`
}

// AuditLogListResult 审计日志分页列表
type AuditLogListResult struct {
	Items    []*AuditLogEntry `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Pages    int              `json:"pages"`
}

// ==================== 上下文 ====================

type contextKey string

const userContextKey contextKey = "user"

// ==================== 服务 ====================

type Service struct {
	db        *sql.DB
	jwtSecret []byte
	logger    *zap.Logger
}

func NewService(db *sql.DB, jwtSecret string, logger *zap.Logger) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

// ==================== 认证 ====================

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return nil, fmt.Errorf("用户名长度需在 3-50 之间")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("密码长度至少 6 位")
	}
	if !strings.Contains(req.Email, "@") {
		return nil, fmt.Errorf("邮箱格式不正确")
	}

	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)",
		req.Username, req.Email,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("数据库查询失败: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 第一个注册的用户自动成为 admin
	role := RoleUser
	var userCount int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		role = RoleAdmin
	} else if req.Role != "" && IsValidRole(req.Role) {
		// 允许注册时选择角色（便于演示多角色功能）
		role = req.Role
	}

	user := &User{}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password_hash, role, status)
		 VALUES ($1, $2, $3, $4, 'active')
		 RETURNING id, username, email, role, status, created_at`,
		req.Username, req.Email, string(hash), role,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	s.logger.Info("User registered", zap.String("username", user.Username), zap.String("role", user.Role))
	return user, nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, status, created_at
		 FROM users WHERE username = $1 AND status = 'active'`,
		req.Username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("数据库查询失败: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	s.db.ExecContext(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", user.ID)

	s.logger.Info("User logged in", zap.String("username", user.Username))

	return &LoginResponse{
		Token:     tokenStr,
		ExpiresAt: expiresAt.Unix(),
		User:      user,
	}, nil
}

// GetUserFromContext 从上下文获取当前用户
func GetUserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// GetCurrentUser 通过 ID 查询当前用户信息
func (s *Service) GetCurrentUser(ctx context.Context, userID int64) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, role, status, created_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// ==================== 中间件 ====================

// AuthMiddleware JWT 认证中间件
func (s *Service) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success":false,"error":"未提供认证令牌"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, `{"success":false,"error":"认证令牌格式错误"}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"success":false,"error":"认证令牌无效或已过期"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"success":false,"error":"令牌解析失败"}`, http.StatusUnauthorized)
			return
		}

		userID := int64(claims["user_id"].(float64))
		user := &User{
			ID:       userID,
			Username: claims["username"].(string),
			Role:     claims["role"].(string),
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole RBAC 角色检查中间件 —— 仅允许指定角色通过
func (s *Service) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"success":false,"error":"未登录"}`, http.StatusUnauthorized)
				return
			}

			// admin 拥有所有权限
			if user.Role == RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"success":false,"error":"权限不足，需要角色: `+strings.Join(roles, ", ")+`"}`, http.StatusForbidden)
		})
	}
}

// ==================== 用户管理 ====================

// ListUsers 分页查询用户列表
func (s *Service) ListUsers(ctx context.Context, role, search string, page, pageSize int) (*UserListResult, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if role != "" && role != "all" {
		where += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}

	if search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// 总数
	var total int
	countQuery := "SELECT COUNT(*) FROM users " + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}

	dataQuery := fmt.Sprintf(
		`SELECT id, username, email, role, status, created_at, last_login_at
		 FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.LastLoginAt); err != nil {
			continue
		}
		users = append(users, &u)
	}
	if users == nil {
		users = []*User{}
	}

	return &UserListResult{
		Items:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}

// UpdateUserRole 修改用户角色
func (s *Service) UpdateUserRole(ctx context.Context, userID int64, newRole string) error {
	if !IsValidRole(newRole) {
		return ErrInvalidRole
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2",
		newRole, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateUserStatus 修改用户状态（启用/禁用）
func (s *Service) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	if status != "active" && status != "disabled" {
		return fmt.Errorf("无效的用户状态: %s", status)
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2",
		status, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetRolesInfo 获取所有角色信息
func (s *Service) GetRolesInfo(ctx context.Context) []map[string]interface{} {
	var result []map[string]interface{}
	for _, r := range AllRoles {
		// 统计每个角色的用户数
		var count int
		s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = $1", r).Scan(&count)
		result = append(result, map[string]interface{}{
			"role":  r,
			"label": RoleLabels[r],
			"count": count,
		})
	}
	return result
}

// ==================== 审计日志 ====================

// RecordAuditLog 记录审计日志
func (s *Service) RecordAuditLog(ctx context.Context, userID *int64, action, resource string, details interface{}, ipAddress string) {
	var detailsJSON *string
	if details != nil {
		b, err := json.Marshal(details)
		if err == nil {
			str := string(b)
			detailsJSON = &str
		}
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (user_id, action, resource, details, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5::inet, NOW())`,
		userID, action, resource, detailsJSON, nilIfEmpty(ipAddress))
	if err != nil {
		s.logger.Error("Failed to record audit log", zap.Error(err), zap.String("action", action))
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListAuditLogs 分页查询审计日志
func (s *Service) ListAuditLogs(ctx context.Context, action, search string, page, pageSize int) (*AuditLogListResult, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if action != "" && action != "all" {
		where += fmt.Sprintf(" AND a.action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}

	if search != "" {
		where += fmt.Sprintf(" AND (a.resource ILIKE $%d OR a.details::text ILIKE $%d OR u.username ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// 总数
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs a LEFT JOIN users u ON a.user_id = u.id %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}

	dataQuery := fmt.Sprintf(
		`SELECT a.id, a.user_id, COALESCE(u.username, '系统'), a.action, 
		        COALESCE(a.resource, ''), COALESCE(a.details::text, ''), 
		        COALESCE(host(a.ip_address), ''), a.created_at
		 FROM audit_logs a
		 LEFT JOIN users u ON a.user_id = u.id
		 %s ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLogEntry
	for rows.Next() {
		var entry AuditLogEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Username, &entry.Action,
			&entry.Resource, &entry.Details, &entry.IPAddress, &entry.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, &entry)
	}
	if logs == nil {
		logs = []*AuditLogEntry{}
	}

	return &AuditLogListResult{
		Items:    logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}

// GetAuditActions 获取所有审计操作类型
func (s *Service) GetAuditActions(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT action FROM audit_logs ORDER BY action")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			continue
		}
		actions = append(actions, action)
	}
	return actions, nil
}
