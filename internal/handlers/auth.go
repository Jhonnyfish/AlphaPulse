package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"alphapulse/internal/config"
	"alphapulse/internal/middleware"
	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db     *pgxpool.Pool
	cfg    *config.Config
	logger *zap.Logger
}

type registerRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createInviteCodeRequest struct {
	MaxUses   *int            `json:"max_uses"`
	ExpiresIn json.RawMessage `json:"expires_in"`
}

type authResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         models.User `json:"user"`
}

func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg, logger: logger}
}

// @Summary      用户注册
// @Description  使用邀请码注册新用户，返回 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerRequest true "注册信息"
// @Success      200 {object} authResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	h.logger.Info("user registration requested")
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	inviteCode := strings.TrimSpace(req.InviteCode)
	if username == "" || password == "" || inviteCode == "" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "username, password, and invite_code are required")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		h.logger.Error("failed to begin transaction for registration", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "TX_START_FAILED", "failed to start transaction")
		return
	}
	defer tx.Rollback(ctx)

	invite, err := fetchInviteCodeForUpdate(ctx, tx, inviteCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusBadRequest, "INVALID_INVITE_CODE", "invalid invite code")
			return
		}
		writeError(c, http.StatusInternalServerError, "INVITE_LOOKUP_FAILED", "failed to validate invite code")
		return
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		writeError(c, http.StatusBadRequest, "INVITE_CODE_EXPIRED", "invite code has expired")
		return
	}
	if invite.Uses >= invite.MaxUses {
		writeError(c, http.StatusBadRequest, "INVITE_CODE_EXHAUSTED", "invite code has reached max uses")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "failed to hash password")
		return
	}

	var user models.User
	err = tx.QueryRow(
		ctx,
		`INSERT INTO users (username, password_hash, role)
		 VALUES ($1, $2, 'user')
		 RETURNING id, username, password_hash, role, created_at`,
		username,
		string(passwordHash),
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			h.logger.Warn("registration failed: username already exists", zap.String("username", username))
			writeError(c, http.StatusBadRequest, "USERNAME_EXISTS", "username already exists")
			return
		}
		h.logger.Error("failed to create user", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "USER_CREATE_FAILED", "failed to create user")
		return
	}

	if _, err := tx.Exec(ctx, `UPDATE invite_codes SET uses = uses + 1 WHERE id = $1`, invite.ID); err != nil {
		h.logger.Error("failed to update invite code uses", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INVITE_UPDATE_FAILED", "failed to update invite code")
		return
	}

	response, err := h.createSession(ctx, tx, user)
	if err != nil {
		h.logger.Error("failed to create session during registration", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "failed to create login session")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		h.logger.Error("failed to commit registration transaction", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "TX_COMMIT_FAILED", "failed to save user")
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      用户登录
// @Description  使用用户名密码登录获取 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "登录信息"
// @Success      200 {object} authResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "username and password are required")
		return
	}

	ctx := c.Request.Context()
	var user models.User
	err := h.db.QueryRow(
		ctx,
		`SELECT id, username, password_hash, role, created_at
		 FROM users
		 WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.Warn("login failed: user not found", zap.String("username", username))
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
			return
		}
		writeError(c, http.StatusInternalServerError, "LOGIN_LOOKUP_FAILED", "failed to load user")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.logger.Warn("login failed: invalid password", zap.String("username", username))
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		h.logger.Error("failed to begin transaction for login", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "TX_START_FAILED", "failed to start transaction")
		return
	}
	defer tx.Rollback(ctx)

	response, err := h.createSession(ctx, tx, user)
	if err != nil {
		h.logger.Error("failed to create session during login", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "failed to create login session")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		h.logger.Error("failed to commit login transaction", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "TX_COMMIT_FAILED", "failed to create session")
		return
	}

	h.logger.Info("user logged in successfully", zap.String("username", username))
	c.JSON(http.StatusOK, response)
}

// @Summary      验证 Token
// @Description  验证当前 JWT token 是否有效，返回用户信息
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Router       /api/auth/verify [get]
func (h *AuthHandler) Verify(c *gin.Context) {
	h.logger.Info("token verification requested")
	user, ok := middleware.CurrentUser(c)
	if !ok {
		h.logger.Warn("token verification failed: no user in context")
		writeError(c, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user":  user,
	})
}

func (h *AuthHandler) CreateInviteCode(c *gin.Context) {
	h.logger.Info("create invite code requested")
	var req createInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	maxUses := 1
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}
	if maxUses < 1 {
		writeError(c, http.StatusBadRequest, "INVALID_MAX_USES", "max_uses must be at least 1")
		return
	}

	expiresAt, err := parseExpiresIn(req.ExpiresIn)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_EXPIRES_IN", "expires_in must be a duration string or number of hours")
		return
	}

	code, err := randomString(12)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "CODE_GENERATION_FAILED", "failed to generate invite code")
		return
	}

	currentUser, _ := middleware.CurrentUser(c)
	ctx := c.Request.Context()
	if _, err := h.db.Exec(
		ctx,
		`INSERT INTO invite_codes (code, created_by, max_uses, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		code,
		currentUser.ID,
		maxUses,
		expiresAt,
	); err != nil {
		h.logger.Error("failed to insert invite code", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INVITE_CREATE_FAILED", "failed to create invite code")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

func (h *AuthHandler) ListInviteCodes(c *gin.Context) {
	h.logger.Info("list invite codes requested")
	rows, err := h.db.Query(
		c.Request.Context(),
		`SELECT id, code, created_by, max_uses, uses, expires_at, created_at
		 FROM invite_codes
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		h.logger.Error("failed to query invite codes", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INVITE_LIST_FAILED", "failed to load invite codes")
		return
	}
	defer rows.Close()

	codes := make([]models.InviteCode, 0)
	for rows.Next() {
		var invite models.InviteCode
		if err := rows.Scan(
			&invite.ID,
			&invite.Code,
			&invite.CreatedBy,
			&invite.MaxUses,
			&invite.Uses,
			&invite.ExpiresAt,
			&invite.CreatedAt,
		); err != nil {
			h.logger.Error("failed to scan invite code row", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "INVITE_SCAN_FAILED", "failed to scan invite code")
			return
		}
		codes = append(codes, invite)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "INVITE_LIST_FAILED", "failed to load invite codes")
		return
	}

	c.JSON(http.StatusOK, codes)
}

func (h *AuthHandler) DeleteInviteCode(c *gin.Context) {
	h.logger.Info("delete invite code requested")
	commandTag, err := h.db.Exec(
		c.Request.Context(),
		`DELETE FROM invite_codes WHERE id = $1`,
		c.Param("id"),
	)
	if err != nil {
		h.logger.Error("failed to delete invite code", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "INVITE_DELETE_FAILED", "failed to delete invite code")
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "INVITE_NOT_FOUND", "invite code not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) createSession(ctx context.Context, tx pgx.Tx, user models.User) (*authResponse, error) {
	authUser := models.AuthUser{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	token, err := middleware.GenerateToken(h.cfg.JWTSecret, authUser, h.cfg.JWTExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := randomString(64)
	if err != nil {
		return nil, err
	}

	refreshExpiry := time.Now().Add(h.cfg.RefreshTokenExpiry)
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		user.ID,
		hashToken(refreshToken),
		refreshExpiry,
	); err != nil {
		return nil, err
	}

	return &authResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func fetchInviteCodeForUpdate(ctx context.Context, tx pgx.Tx, code string) (*models.InviteCode, error) {
	var invite models.InviteCode
	err := tx.QueryRow(
		ctx,
		`SELECT id, code, created_by, max_uses, uses, expires_at, created_at
		 FROM invite_codes
		 WHERE code = $1
		 FOR UPDATE`,
		code,
	).Scan(
		&invite.ID,
		&invite.Code,
		&invite.CreatedBy,
		&invite.MaxUses,
		&invite.Uses,
		&invite.ExpiresAt,
		&invite.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &invite, nil
}

func parseExpiresIn(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var durationText string
	if err := json.Unmarshal(raw, &durationText); err == nil {
		if strings.TrimSpace(durationText) == "" {
			return nil, nil
		}
		duration, err := time.ParseDuration(durationText)
		if err != nil {
			return nil, err
		}
		expiresAt := time.Now().Add(duration)
		return &expiresAt, nil
	}

	var hours float64
	if err := json.Unmarshal(raw, &hours); err == nil {
		if hours <= 0 {
			return nil, nil
		}
		expiresAt := time.Now().Add(time.Duration(hours * float64(time.Hour)))
		return &expiresAt, nil
	}

	return nil, fmt.Errorf("invalid expires_in value")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
