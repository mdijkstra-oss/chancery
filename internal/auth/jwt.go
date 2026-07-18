package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	JWKSURL       string
	PublicKeyFile string
	Issuer        string
	Audience      string
	Algorithms    []string
}

type Validator interface {
	Close()
	Enabled() bool
	Validate(context.Context, string) (string, error)
}

func NewValidator(ctx context.Context, cfg Config) (Validator, error) {
	return newValidator(ctx, cfg, nil)
}

func (c Config) Enabled() bool {
	return c.JWKSURL != "" || c.PublicKeyFile != ""
}

func (c Config) Validate() error {
	if c.JWKSURL != "" && c.PublicKeyFile != "" {
		return errors.New("AUTH_JWT_JWKS_URL and AUTH_JWT_PUBLIC_KEY_FILE are mutually exclusive")
	}
	if !c.Enabled() {
		return nil
	}
	if c.Issuer == "" {
		return errors.New("AUTH_JWT_ISSUER is required when auth is enabled")
	}
	if c.Audience == "" {
		return errors.New("AUTH_JWT_AUDIENCE is required when auth is enabled")
	}
	if len(c.Algorithms) == 0 {
		return errors.New("AUTH_JWT_ALGORITHMS is required when auth is enabled")
	}
	if c.JWKSURL != "" {
		if err := validateJWKSURL(c.JWKSURL); err != nil {
			return err
		}
	}
	for _, algorithm := range c.Algorithms {
		if !isAsymmetricAlgorithm(algorithm) {
			return fmt.Errorf("unsupported JWT algorithm %q", algorithm)
		}
	}
	return nil
}

func UserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey{}).(string)
	return user
}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func newValidator(ctx context.Context, cfg Config, client *http.Client) (Validator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if !cfg.Enabled() {
		return disabledValidator{}, nil
	}
	parser := jwt.NewParser(
		jwt.WithAudience(cfg.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithLeeway(30*time.Second),
		jwt.WithStrictDecoding(),
		jwt.WithValidMethods(cfg.Algorithms),
	)
	if cfg.PublicKeyFile != "" {
		key, err := loadPublicKey(cfg.PublicKeyFile)
		if err != nil {
			return nil, err
		}
		if err := validateKeyAlgorithms(key, cfg.Algorithms); err != nil {
			return nil, err
		}
		return &jwtValidator{parser: parser, keyfunc: staticKeyfunc(key), cancel: func() {}}, nil
	}
	cacheCtx, cancel := context.WithCancel(ctx)
	override := keyfunc.Override{HTTPTimeout: 10 * time.Second}
	if client != nil {
		override.Client = client
	}
	remote, err := keyfunc.NewDefaultOverrideCtx(cacheCtx, []string{cfg.JWKSURL}, override)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("load JWKS: %w", err)
	}
	return &jwtValidator{parser: parser, keyfunc: remote.KeyfuncCtx(cacheCtx), cancel: cancel}, nil
}

type userContextKey struct{}

type disabledValidator struct{}

func (disabledValidator) Close() {}

func (disabledValidator) Enabled() bool {
	return false
}

func (disabledValidator) Validate(context.Context, string) (string, error) {
	return "", nil
}

type jwtValidator struct {
	parser  *jwt.Parser
	keyfunc jwt.Keyfunc
	cancel  context.CancelFunc
}

func (v *jwtValidator) Close() {
	v.cancel()
}

func (v *jwtValidator) Enabled() bool {
	return true
}

func (v *jwtValidator) Validate(_ context.Context, raw string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := v.parser.ParseWithClaims(raw, claims, v.keyfunc)
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}
	subject := claims.Subject
	if strings.TrimSpace(subject) == "" {
		return "", fmt.Errorf("%w: sub", jwt.ErrTokenRequiredClaimMissing)
	}
	if len(subject) > 256 || strings.ContainsAny(subject, "\r\n") {
		return "", fmt.Errorf("%w: sub", jwt.ErrTokenInvalidSubject)
	}
	return subject, nil
}

func staticKeyfunc(key any) jwt.Keyfunc {
	return func(*jwt.Token) (any, error) {
		return key, nil
	}
}

func loadPublicKey(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read JWT public key: %w", err)
	}
	parsers := []func([]byte) (any, error){
		parseRSAPublicKey,
		parseECDSAPublicKey,
		parseEdPublicKey,
	}
	for _, parse := range parsers {
		key, parseErr := parse(data)
		if parseErr == nil {
			return key, nil
		}
	}
	return nil, errors.New("parse JWT public key: unsupported PEM public key")
}

func parseRSAPublicKey(data []byte) (any, error) {
	return jwt.ParseRSAPublicKeyFromPEM(data)
}

func parseECDSAPublicKey(data []byte) (any, error) {
	return jwt.ParseECPublicKeyFromPEM(data)
}

func parseEdPublicKey(data []byte) (any, error) {
	return jwt.ParseEdPublicKeyFromPEM(data)
}

func validateKeyAlgorithms(key any, algorithms []string) error {
	if rsaKey, ok := key.(*rsa.PublicKey); ok && rsaKey.N.BitLen() < 2048 {
		return errors.New("JWT RSA public key must be at least 2048 bits")
	}
	for _, algorithm := range algorithms {
		if !keySupportsAlgorithm(key, algorithm) {
			return fmt.Errorf("JWT public key does not support algorithm %q", algorithm)
		}
	}
	return nil
}

func keySupportsAlgorithm(key any, algorithm string) bool {
	switch typed := key.(type) {
	case *rsa.PublicKey:
		return strings.HasPrefix(algorithm, "RS") || strings.HasPrefix(algorithm, "PS")
	case *ecdsa.PublicKey:
		return ecdsaAlgorithm(typed) == algorithm
	case ed25519.PublicKey:
		return algorithm == jwt.SigningMethodEdDSA.Alg()
	default:
		panic(fmt.Sprintf("unknown JWT public key type %T", key))
	}
}

func ecdsaAlgorithm(key *ecdsa.PublicKey) string {
	switch key.Curve.Params().BitSize {
	case 256:
		return "ES256"
	case 384:
		return "ES384"
	case 521:
		return "ES512"
	default:
		panic(fmt.Sprintf("unknown JWT ECDSA curve size %d", key.Curve.Params().BitSize))
	}
}

func validateJWKSURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("AUTH_JWT_JWKS_URL must be an HTTPS URL without credentials, query, or fragment")
	}
	return nil
}

func isAsymmetricAlgorithm(algorithm string) bool {
	switch algorithm {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EdDSA":
		return true
	default:
		return false
	}
}
