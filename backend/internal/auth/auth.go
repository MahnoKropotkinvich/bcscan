package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists         = errors.New("用户名或邮箱已存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
)

// User 用户模型
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // 不返回给前端
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      *User  `json:"user"`
}

// contextKey 上下文键
type contextKey string

const userContextKey contextKey = "user"

// Service 认证服务
type Service struct {
	db        *sql.DB
	jwtSecret []byte
	logger    *zap.Logger
}

// NewService 创建认证服务
func NewService(db *sql.DB, jwtSecret string, logger *zap.Logger) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	// 验证输入
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return nil, fmt.Errorf("用户名长度需在 3-50 之间")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("密码长度至少 6 位")
	}
	if !strings.Contains(req.Email, "@") {
		return nil, fmt.Errorf("邮箱格式不正确")
	}

	// 检查用户名/邮箱是否已存在
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

	// bcrypt 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 插入用户
	user := &User{}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password_hash, role, status)
		 VALUES ($1, $2, $3, 'user', 'active')
		 RETURNING id, username, email, role, status, created_at`,
		req.Username, req.Email, string(hash),
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	s.logger.Info("User registered", zap.String("username", user.Username))
	return user, nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 查询用户
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

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 生成 JWT
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

	// 更新最后登录时间
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

// GetCurrentUser 通过 token 查询当前用户信息
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

// AuthMiddleware JWT 认证中间件
func (s *Service) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从 Header 提取 Token
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

		// 解析并验证 JWT
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

		// 从 claims 构建 User 放入 context
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
