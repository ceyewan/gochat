package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	logicpb "github.com/ceyewan/gochat/api/gen/im_logic/v1"
	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthService 认证服务
type AuthService struct {
	config    *config.Config
	logger    clog.Logger
	client    *grpc.Client
	jwtSecret []byte
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, client *grpc.Client) *AuthService {
	logger := clog.Namespace("auth-service")

	return &AuthService{
		config:    cfg,
		logger:    logger,
		client:    client,
		jwtSecret: []byte(cfg.JWT.Secret),
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *logicpb.LoginRequest) (*logicpb.LoginResponse, error) {
	s.logger.Info("用户登录", clog.String("username", req.Username))

	// 验证参数
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名和密码不能为空")
	}

	// 获取用户信息
	userRepo := s.client.GetUserServiceClient()
	userResp, err := s.client.RetryWithBackoff(ctx, func(ctx context.Context) error {
		userReq := &repopb.GetUserByUsernameRequest{
			Username: req.Username,
		}
		_, err := userRepo.GetUserByUsername(ctx, userReq)
		return err
	})

	if err != nil {
		s.logger.Error("获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "用户不存在")
	}

	// 获取用户详细信息
	userReq := &repopb.GetUserByUsernameRequest{
		Username: req.Username,
	}
	userInfo, err := userRepo.GetUserByUsername(ctx, userReq)
	if err != nil {
		s.logger.Error("获取用户详细信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户信息失败")
	}

	// 验证密码
	verifyReq := &repopb.VerifyPasswordRequest{
		UserId:   userInfo.User.Id,
		Password: req.Password,
	}
	verifyResp, err := userRepo.VerifyPassword(ctx, verifyReq)
	if err != nil {
		s.logger.Error("验证密码失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "验证密码失败")
	}

	if !verifyResp.Valid {
		s.logger.Warn("密码验证失败", clog.String("username", req.Username))
		return nil, status.Error(codes.Unauthenticated, "用户名或密码错误")
	}

	// 生成令牌
	accessToken, refreshToken, err := s.generateTokens(userInfo.User.Id)
	if err != nil {
		s.logger.Error("生成令牌失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "生成令牌失败")
	}

	// 转换用户信息
	user := &logicpb.User{
		Id:        userInfo.User.Id,
		Username:  userInfo.User.Username,
		Nickname:  userInfo.User.Nickname,
		AvatarUrl: userInfo.User.AvatarUrl,
		CreatedAt: userInfo.User.CreatedAt,
		UpdatedAt: userInfo.User.UpdatedAt,
	}

	s.logger.Info("用户登录成功", clog.String("username", req.Username), clog.String("user_id", userInfo.User.Id))

	return &logicpb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Now().Add(s.config.GetAccessTokenExpireDuration()).Unix(),
		User:         user,
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *logicpb.RegisterRequest) (*logicpb.RegisterResponse, error) {
	s.logger.Info("用户注册", clog.String("username", req.Username))

	// 验证参数
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名和密码不能为空")
	}

	// 检查用户名是否已存在
	userRepo := s.client.GetUserServiceClient()
	_, err := userRepo.GetUserByUsername(ctx, &repopb.GetUserByUsernameRequest{
		Username: req.Username,
	})
	if err == nil {
		s.logger.Warn("用户名已存在", clog.String("username", req.Username))
		return nil, status.Error(codes.AlreadyExists, "用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("加密密码失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "注册失败")
	}

	// 创建用户
	createReq := &repopb.CreateUserRequest{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Nickname:     req.Nickname,
		AvatarUrl:    req.AvatarUrl,
	}

	createResp, err := userRepo.CreateUser(ctx, createReq)
	if err != nil {
		s.logger.Error("创建用户失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "注册失败")
	}

	// 转换用户信息
	user := &logicpb.User{
		Id:        createResp.User.Id,
		Username:  createResp.User.Username,
		Nickname:  createResp.User.Nickname,
		AvatarUrl: createResp.User.AvatarUrl,
		CreatedAt: createResp.User.CreatedAt,
		UpdatedAt: createResp.User.UpdatedAt,
	}

	s.logger.Info("用户注册成功", clog.String("username", req.Username), clog.String("user_id", createResp.User.Id))

	return &logicpb.RegisterResponse{
		User: user,
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(ctx context.Context, req *logicpb.RefreshTokenRequest) (*logicpb.RefreshTokenResponse, error) {
	s.logger.Info("刷新访问令牌")

	// 验证刷新令牌
	claims, err := s.validateToken(req.RefreshToken)
	if err != nil {
		s.logger.Error("刷新令牌验证失败", clog.Err(err))
		return nil, status.Error(codes.Unauthenticated, "无效的刷新令牌")
	}

	// 检查令牌类型
	if claims.Type != "refresh" {
		s.logger.Warn("令牌类型错误", clog.String("type", claims.Type))
		return nil, status.Error(codes.Unauthenticated, "无效的令牌类型")
	}

	// 获取用户信息
	userRepo := s.client.GetUserServiceClient()
	userResp, err := userRepo.GetUser(ctx, &repopb.GetUserRequest{
		UserId: claims.UserID,
	})
	if err != nil {
		s.logger.Error("获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "用户不存在")
	}

	// 生成新的令牌
	accessToken, refreshToken, err := s.generateTokens(userResp.User.Id)
	if err != nil {
		s.logger.Error("生成令牌失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "生成令牌失败")
	}

	s.logger.Info("访问令牌刷新成功", clog.String("user_id", claims.UserID))

	return &logicpb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Now().Add(s.config.GetAccessTokenExpireDuration()).Unix(),
	}, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, req *logicpb.LogoutRequest) (*logicpb.LogoutResponse, error) {
	s.logger.Info("用户登出", clog.String("user_id", req.UserId))

	// 验证访问令牌
	_, err := s.validateToken(req.AccessToken)
	if err != nil {
		s.logger.Error("访问令牌验证失败", clog.Err(err))
		return nil, status.Error(codes.Unauthenticated, "无效的访问令牌")
	}

	// TODO: 将令牌加入黑名单
	// 这里可以实现将令牌加入 Redis 黑名单，使其失效

	s.logger.Info("用户登出成功", clog.String("user_id", req.UserId))

	return &logicpb.LogoutResponse{
		Success: true,
	}, nil
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(ctx context.Context, req *logicpb.ValidateTokenRequest) (*logicpb.ValidateTokenResponse, error) {
	s.logger.Info("验证访问令牌")

	// 验证令牌
	claims, err := s.validateToken(req.AccessToken)
	if err != nil {
		s.logger.Error("令牌验证失败", clog.Err(err))
		return &logicpb.ValidateTokenResponse{
			Valid:     false,
			ExpiresAt: 0,
		}, nil
	}

	// 检查令牌类型
	if claims.Type != "access" {
		s.logger.Warn("令牌类型错误", clog.String("type", claims.Type))
		return &logicpb.ValidateTokenResponse{
			Valid:     false,
			ExpiresAt: 0,
		}, nil
	}

	// 获取用户信息
	userRepo := s.client.GetUserServiceClient()
	userResp, err := userRepo.GetUser(ctx, &repopb.GetUserRequest{
		UserId: claims.UserID,
	})
	if err != nil {
		s.logger.Error("获取用户信息失败", clog.Err(err))
		return &logicpb.ValidateTokenResponse{
			Valid:     false,
			ExpiresAt: 0,
		}, nil
	}

	// 转换用户信息
	user := &logicpb.User{
		Id:        userResp.User.Id,
		Username:  userResp.User.Username,
		Nickname:  userResp.User.Nickname,
		AvatarUrl: userResp.User.AvatarUrl,
		CreatedAt: userResp.User.CreatedAt,
		UpdatedAt: userResp.User.UpdatedAt,
	}

	s.logger.Info("令牌验证成功", clog.String("user_id", claims.UserID))

	return &logicpb.ValidateTokenResponse{
		Valid:     true,
		User:      user,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

// generateTokens 生成访问令牌和刷新令牌
func (s *AuthService) generateTokens(userID string) (string, string, error) {
	now := time.Now()

	// 生成访问令牌
	accessToken, err := s.generateToken(userID, "access", s.config.GetAccessTokenExpireDuration())
	if err != nil {
		return "", "", err
	}

	// 生成刷新令牌
	refreshToken, err := s.generateToken(userID, "refresh", s.config.GetRefreshTokenExpireDuration())
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// generateToken 生成 JWT 令牌
func (s *AuthService) generateToken(userID, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			Issuer:    "im-logic",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(s.config.JWT.SigningMethod), claims)
	return token.SignedString(s.jwtSecret)
}

// validateToken 验证 JWT 令牌
func (s *AuthService) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.GetSigningMethod(s.config.JWT.SigningMethod) {
			return nil, fmt.Errorf("无效的签名方法")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的令牌")
}

// JWTClaims JWT 令牌声明
type JWTClaims struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

// HashPassword 哈希密码
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword 检查密码
func (s *AuthService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateSHA256 生成 SHA256 哈希
func (s *AuthService) GenerateSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:])
}
