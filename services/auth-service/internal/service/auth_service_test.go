package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"task-manager/auth-service/internal/model"
	"task-manager/auth-service/internal/service"
)

const testSecret = "test-secret-key"

// mockUserRepo is a testify mock for repository.UserRepository.
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(email, passwordHash string) (*model.User, error) {
	args := m.Called(email, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) GetByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	repo.On("GetByEmail", "alice@example.com").Return(nil, nil)
	repo.On("Create", "alice@example.com", mock.AnythingOfType("string")).
		Return(&model.User{ID: "u1", Email: "alice@example.com"}, nil)

	user, err := svc.Register(model.RegisterRequest{Email: "alice@example.com", Password: "secret"})
	assert.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	repo.AssertExpectations(t)
}

func TestRegister_RejectsEmptyFields(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	_, err := svc.Register(model.RegisterRequest{Email: "", Password: "secret"})
	assert.Error(t, err)

	_, err = svc.Register(model.RegisterRequest{Email: "a@b.com", Password: ""})
	assert.Error(t, err)

	repo.AssertNotCalled(t, "GetByEmail")
}

func TestRegister_ReturnsErrUserExistsWhenDuplicate(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	repo.On("GetByEmail", "dup@example.com").
		Return(&model.User{ID: "u2", Email: "dup@example.com"}, nil)

	_, err := svc.Register(model.RegisterRequest{Email: "dup@example.com", Password: "pass"})
	assert.ErrorIs(t, err, service.ErrUserExists)
	repo.AssertExpectations(t)
}

func TestRegister_ReturnsErrorOnRepoFailure(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	repo.On("GetByEmail", "a@b.com").Return(nil, errors.New("db error"))

	_, err := svc.Register(model.RegisterRequest{Email: "a@b.com", Password: "pass"})
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	hash, _ := bcrypt.GenerateFromPassword([]byte("mypassword"), bcrypt.MinCost)
	repo.On("GetByEmail", "bob@example.com").
		Return(&model.User{ID: "u3", Email: "bob@example.com", PasswordHash: string(hash)}, nil)

	token, err := svc.Login(model.LoginRequest{Email: "bob@example.com", Password: "mypassword"})
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	repo.AssertExpectations(t)
}

func TestLogin_ReturnsErrInvalidCredsWhenUserNotFound(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	repo.On("GetByEmail", "nobody@example.com").Return(nil, nil)

	_, err := svc.Login(model.LoginRequest{Email: "nobody@example.com", Password: "pass"})
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
	repo.AssertExpectations(t)
}

func TestLogin_ReturnsErrInvalidCredsOnWrongPassword(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	repo.On("GetByEmail", "carol@example.com").
		Return(&model.User{ID: "u4", Email: "carol@example.com", PasswordHash: string(hash)}, nil)

	_, err := svc.Login(model.LoginRequest{Email: "carol@example.com", Password: "wrong"})
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
	repo.AssertExpectations(t)
}

func TestLogin_RejectsEmptyFields(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	_, err := svc.Login(model.LoginRequest{Email: "", Password: "pass"})
	assert.Error(t, err)
	repo.AssertNotCalled(t, "GetByEmail")
}

// --- ValidateToken ---

func TestValidateToken_RoundTrip(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	repo.On("GetByEmail", "dave@example.com").
		Return(&model.User{ID: "u5", Email: "dave@example.com", PasswordHash: string(hash)}, nil)

	token, err := svc.Login(model.LoginRequest{Email: "dave@example.com", Password: "pw"})
	assert.NoError(t, err)

	userID, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "u5", userID)
}

func TestValidateToken_RejectsGarbage(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewAuthService(repo, testSecret)

	_, err := svc.ValidateToken("not.a.jwt")
	assert.ErrorIs(t, err, service.ErrInvalidToken)
}

func TestValidateToken_RejectsWrongSecret(t *testing.T) {
	repo := &mockUserRepo{}
	svcA := service.NewAuthService(repo, "secret-a")
	svcB := service.NewAuthService(repo, "secret-b")

	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	repo.On("GetByEmail", "eve@example.com").
		Return(&model.User{ID: "u6", Email: "eve@example.com", PasswordHash: string(hash)}, nil)

	token, err := svcA.Login(model.LoginRequest{Email: "eve@example.com", Password: "pw"})
	assert.NoError(t, err)

	_, err = svcB.ValidateToken(token)
	assert.ErrorIs(t, err, service.ErrInvalidToken)
}
