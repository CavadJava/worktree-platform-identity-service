package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/database"
	"platform-identity-service/internal/models"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestShopRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewShopRepository(db)

	s := &models.Shop{ID: uuid.NewString(), Name: "Test Shop " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != s.Name {
		t.Errorf("expected name %q, got %q", s.Name, got.Name)
	}
}

func TestShopRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewShopRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if err != ErrShopNotFound {
		t.Errorf("expected ErrShopNotFound, got %v", err)
	}
}

func TestSystemRoleRepository_List(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewSystemRoleRepository(db)

	roles, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("expected 3 seeded system roles, got %d", len(roles))
	}
	byName := map[string]int16{}
	for _, r := range roles {
		byName[r.Name] = r.ID
	}
	if byName["superadmin"] != 1 || byName["admin"] != 2 || byName["user"] != 3 {
		t.Errorf("unexpected system role ids: %+v", byName)
	}
}

func TestShopRoleRepository_List(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewShopRoleRepository(db)

	roles, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 seeded shop roles, got %d", len(roles))
	}
	byName := map[string]int16{}
	for _, r := range roles {
		byName[r.Name] = r.ID
	}
	if byName["shop-admin"] != 1 || byName["shop-user"] != 2 {
		t.Errorf("unexpected shop role ids: %+v", byName)
	}
}

func createTestUser(t *testing.T, userRepo *UserRepository, systemRoleID int16) *models.User {
	t.Helper()
	now := time.Now().UTC()
	u := &models.User{
		ID: uuid.NewString(), Name: "Test User", Username: "u-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		SystemRoleID: systemRoleID, Status: models.UserStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return u
}

func TestUserRepository_CreateGetListAll(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)

	u := createTestUser(t, userRepo, 3) // system role 'user'

	got, err := userRepo.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.SystemRoleName != "user" {
		t.Errorf("expected SystemRoleName 'user', got %q", got.SystemRoleName)
	}
	if got.Status != models.UserStatusActive {
		t.Errorf("expected status ACTIVE, got %q", got.Status)
	}

	all, err := userRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	found := false
	for _, u2 := range all {
		if u2.ID == u.ID {
			found = true
		}
	}
	if !found {
		t.Error("expected created user in ListAll result")
	}
}

func TestUserRepository_GetByUsernameOrEmail(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)

	u := createTestUser(t, userRepo, 3)

	got, err := userRepo.GetByUsernameOrEmail(context.Background(), u.Username)
	if err != nil {
		t.Fatalf("by username failed: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected id %q, got %q", u.ID, got.ID)
	}

	got, err = userRepo.GetByUsernameOrEmail(context.Background(), u.Email)
	if err != nil {
		t.Fatalf("by email failed: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected id %q, got %q", u.ID, got.ID)
	}

	_, err = userRepo.GetByUsernameOrEmail(context.Background(), "nonexistent-"+uuid.NewString())
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_SetSystemRole(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)

	u := createTestUser(t, userRepo, 3)

	if err := userRepo.SetSystemRole(context.Background(), u.ID, 2); err != nil {
		t.Fatalf("SetSystemRole failed: %v", err)
	}

	got, err := userRepo.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.SystemRoleName != "admin" {
		t.Errorf("expected 'admin', got %q", got.SystemRoleName)
	}
}

func TestUserRepository_SetStatus(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)

	u := createTestUser(t, userRepo, 3)

	if err := userRepo.SetStatus(context.Background(), u.ID, models.UserStatusInActive); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	got, err := userRepo.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Status != models.UserStatusInActive {
		t.Errorf("expected IN_ACTIVE, got %q", got.Status)
	}
}

func TestShopMembershipRepository_CreateListGetSetRole(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	shopRepo := NewShopRepository(db)
	userRepo := NewUserRepository(db)
	membershipRepo := NewShopMembershipRepository(db)

	shop := &models.Shop{ID: uuid.NewString(), Name: "Membership Shop " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := shopRepo.Create(context.Background(), shop); err != nil {
		t.Fatalf("create shop failed: %v", err)
	}
	u1 := createTestUser(t, userRepo, 3)
	u2 := createTestUser(t, userRepo, 3)

	m1 := &models.ShopMembership{ID: uuid.NewString(), UserID: u1.ID, ShopID: shop.ID, ShopRoleID: 1, CreatedAt: time.Now().UTC()}
	if err := membershipRepo.Create(context.Background(), m1); err != nil {
		t.Fatalf("create membership 1 failed: %v", err)
	}
	m2 := &models.ShopMembership{ID: uuid.NewString(), UserID: u2.ID, ShopID: shop.ID, ShopRoleID: 2, CreatedAt: time.Now().UTC()}
	if err := membershipRepo.Create(context.Background(), m2); err != nil {
		t.Fatalf("create membership 2 failed: %v", err)
	}

	// Duplicate membership should fail.
	dup := &models.ShopMembership{ID: uuid.NewString(), UserID: u1.ID, ShopID: shop.ID, ShopRoleID: 2, CreatedAt: time.Now().UTC()}
	if err := membershipRepo.Create(context.Background(), dup); err != ErrMembershipExists {
		t.Errorf("expected ErrMembershipExists, got %v", err)
	}

	members, err := membershipRepo.ListByShop(context.Background(), shop.ID)
	if err != nil {
		t.Fatalf("ListByShop failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	got, err := membershipRepo.GetByUserAndShop(context.Background(), u1.ID, shop.ID)
	if err != nil {
		t.Fatalf("GetByUserAndShop failed: %v", err)
	}
	if got.ShopRoleName != "shop-admin" {
		t.Errorf("expected shop-admin, got %q", got.ShopRoleName)
	}
	if got.ShopName != shop.Name {
		t.Errorf("expected joined ShopName %q, got %q", shop.Name, got.ShopName)
	}

	if err := membershipRepo.SetShopRole(context.Background(), u1.ID, shop.ID, 2); err != nil {
		t.Fatalf("SetShopRole failed: %v", err)
	}
	got, err = membershipRepo.GetByUserAndShop(context.Background(), u1.ID, shop.ID)
	if err != nil {
		t.Fatalf("GetByUserAndShop after SetShopRole failed: %v", err)
	}
	if got.ShopRoleName != "shop-user" {
		t.Errorf("expected shop-user after change, got %q", got.ShopRoleName)
	}

	byUser, err := membershipRepo.ListByUser(context.Background(), u1.ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(byUser) != 1 || byUser[0].ShopID != shop.ID {
		t.Errorf("expected 1 membership for u1 in shop %q, got %+v", shop.ID, byUser)
	}
}

func TestProductRepository_CreateListGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewProductRepository(db)

	p := &models.Product{ID: uuid.NewString(), Name: "Test Product " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != p.Name {
		t.Errorf("expected name %q, got %q", p.Name, got.Name)
	}

	all, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	found := false
	for _, p2 := range all {
		if p2.ID == p.ID {
			found = true
		}
	}
	if !found {
		t.Error("expected created product in List result")
	}
}

func TestSubscriptionRepository_UpsertAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)
	productRepo := NewProductRepository(db)
	subRepo := NewSubscriptionRepository(db)

	u := createTestUser(t, userRepo, 3)
	p := &models.Product{ID: uuid.NewString(), Name: "Sub Product " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := productRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	now := time.Now().UTC()
	sub := &models.Subscription{ID: uuid.NewString(), UserID: u.ID, ProductID: p.ID, Subscripted: false, Renewed: false, CreatedAt: now, UpdatedAt: now}
	if err := subRepo.Upsert(context.Background(), sub); err != nil {
		t.Fatalf("initial upsert failed: %v", err)
	}

	got, err := subRepo.GetByUserAndProduct(context.Background(), u.ID, p.ID)
	if err != nil {
		t.Fatalf("GetByUserAndProduct failed: %v", err)
	}
	if got.Subscripted {
		t.Error("expected Subscripted=false initially")
	}

	sub.Subscripted = true
	sub.UpdatedAt = time.Now().UTC()
	if err := subRepo.Upsert(context.Background(), sub); err != nil {
		t.Fatalf("update upsert failed: %v", err)
	}

	got, err = subRepo.GetByUserAndProduct(context.Background(), u.ID, p.ID)
	if err != nil {
		t.Fatalf("GetByUserAndProduct after update failed: %v", err)
	}
	if !got.Subscripted {
		t.Error("expected Subscripted=true after upsert update")
	}

	_, err = subRepo.GetByUserAndProduct(context.Background(), u.ID, uuid.NewString())
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}
