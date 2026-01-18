package data

import (
	"context"
	"testing"

	"github.com/airplne/calendar-app/server/internal/domain"
)

func TestSQLiteCalendarRepo_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "personal",
		DisplayName: "Personal Calendar",
		Color:       "#FF5733",
		Description: "My personal events",
	}

	// Create
	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if calendar.ID == 0 {
		t.Error("Calendar ID should be set after create")
	}

	// Get by ID
	got, err := repo.GetByID(ctx, calendar.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != calendar.Name {
		t.Errorf("Name mismatch: got %s, want %s", got.Name, calendar.Name)
	}
	if got.DisplayName != calendar.DisplayName {
		t.Errorf("DisplayName mismatch: got %s, want %s", got.DisplayName, calendar.DisplayName)
	}
	if got.Color != calendar.Color {
		t.Errorf("Color mismatch: got %s, want %s", got.Color, calendar.Color)
	}
	if got.Description != calendar.Description {
		t.Errorf("Description mismatch: got %s, want %s", got.Description, calendar.Description)
	}
	if got.UserID != calendar.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", got.UserID, calendar.UserID)
	}
}

func TestSQLiteCalendarRepo_Create_Conflict(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "duplicate",
		DisplayName: "Duplicate Calendar",
	}

	// Create first calendar
	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Try to create duplicate (same user_id + name)
	calendar2 := &domain.Calendar{
		UserID:      userID,
		Name:        "duplicate",
		DisplayName: "Another Duplicate",
	}

	err := repo.Create(ctx, calendar2)
	if err != domain.ErrConflict {
		t.Errorf("Expected ErrConflict, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_Create_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		calendar *domain.Calendar
	}{
		{
			name: "missing name",
			calendar: &domain.Calendar{
				UserID:      userID,
				Name:        "", // Invalid
				DisplayName: "Test",
			},
		},
		{
			name: "invalid user ID",
			calendar: &domain.Calendar{
				UserID:      0, // Invalid
				Name:        "test",
				DisplayName: "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.calendar)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

func TestSQLiteCalendarRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_GetByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "work",
		DisplayName: "Work Calendar",
	}

	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by name
	got, err := repo.GetByName(ctx, userID, "work")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if got.ID != calendar.ID {
		t.Errorf("ID mismatch: got %d, want %d", got.ID, calendar.ID)
	}
	if got.Name != "work" {
		t.Errorf("Name mismatch: got %s, want %s", got.Name, "work")
	}
}

func TestSQLiteCalendarRepo_GetByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, userID, "non-existent")
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user1ID := createTestUser(t, db)

	// Create another user
	userRepo := NewSQLiteUserRepo(db)
	user2, _ := userRepo.Create(context.Background(), "testuser2")
	user2ID := user2.ID

	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	// Create calendars for user1
	calendars := []*domain.Calendar{
		{UserID: user1ID, Name: "personal", DisplayName: "Personal"},
		{UserID: user1ID, Name: "work", DisplayName: "Work"},
		{UserID: user2ID, Name: "other", DisplayName: "Other User"},
	}

	for _, cal := range calendars {
		if err := repo.Create(ctx, cal); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List calendars for user1
	list, err := repo.ListByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}

	// Should only return 2 calendars for user1
	if len(list) != 2 {
		t.Errorf("Expected 2 calendars for user1, got %d", len(list))
	}

	// Verify all returned calendars belong to user1
	for _, cal := range list {
		if cal.UserID != user1ID {
			t.Errorf("Calendar %s belongs to wrong user: %d", cal.Name, cal.UserID)
		}
	}
}

func TestSQLiteCalendarRepo_ListByUser_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	list, err := repo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected 0 calendars, got %d", len(list))
	}
}

func TestSQLiteCalendarRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "personal",
		DisplayName: "Personal Calendar",
		Color:       "#FF5733",
	}

	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	calendar.DisplayName = "Updated Personal Calendar"
	calendar.Color = "#00FF00"
	calendar.Description = "New description"

	if err := repo.Update(ctx, calendar); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	got, _ := repo.GetByID(ctx, calendar.ID)
	if got.DisplayName != "Updated Personal Calendar" {
		t.Errorf("DisplayName not updated: got %s", got.DisplayName)
	}
	if got.Color != "#00FF00" {
		t.Errorf("Color not updated: got %s", got.Color)
	}
	if got.Description != "New description" {
		t.Errorf("Description not updated: got %s", got.Description)
	}
}

func TestSQLiteCalendarRepo_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		ID:          99999,
		UserID:      userID,
		Name:        "test",
		DisplayName: "Test",
	}

	err := repo.Update(ctx, calendar)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "delete-me",
		DisplayName: "Delete Me",
	}

	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, calendar.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := repo.GetByID(ctx, calendar.ID)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteCalendarRepo_IncrementSyncToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	calendar := &domain.Calendar{
		UserID:      userID,
		Name:        "sync-test",
		DisplayName: "Sync Test",
	}

	if err := repo.Create(ctx, calendar); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Initial sync token should be empty
	got, _ := repo.GetByID(ctx, calendar.ID)
	initialToken := got.SyncToken

	// Increment sync token
	token1, err := repo.IncrementSyncToken(ctx, calendar.ID)
	if err != nil {
		t.Fatalf("IncrementSyncToken failed: %v", err)
	}

	if token1 == "" {
		t.Error("Sync token should not be empty")
	}
	if token1 == initialToken {
		t.Error("Sync token should change after increment")
	}

	// Increment again
	token2, err := repo.IncrementSyncToken(ctx, calendar.ID)
	if err != nil {
		t.Fatalf("Second IncrementSyncToken failed: %v", err)
	}

	if token2 == token1 {
		t.Error("Sync token should change on each increment")
	}

	// Verify token is persisted
	got, _ = repo.GetByID(ctx, calendar.ID)
	if got.SyncToken != token2 {
		t.Errorf("Sync token not persisted: got %s, want %s", got.SyncToken, token2)
	}
}

func TestSQLiteCalendarRepo_IncrementSyncToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteCalendarRepo(db)
	ctx := context.Background()

	_, err := repo.IncrementSyncToken(ctx, 99999)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}
