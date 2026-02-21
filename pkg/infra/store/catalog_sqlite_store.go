package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/catalog"
)

// CatalogSQLiteStore implements catalog.RecipeStore using SQLite.
// Recipes are serialised as JSON blobs so that schema changes to the
// Recipe struct don't require a migration.
type CatalogSQLiteStore struct {
	db *sql.DB
}

// NewCatalogSQLiteStore creates a new SQLite-backed catalog recipe store.
func NewCatalogSQLiteStore(db *sql.DB) (*CatalogSQLiteStore, error) {
	s := &CatalogSQLiteStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
}

func (s *CatalogSQLiteStore) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS catalog_recipes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		gpu_vendor TEXT NOT NULL DEFAULT '',
		verified INTEGER NOT NULL DEFAULT 0,
		data TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_recipes_gpu_vendor ON catalog_recipes(gpu_vendor);
	CREATE INDEX IF NOT EXISTS idx_recipes_verified ON catalog_recipes(verified);
	`
	_, err := s.db.Exec(query)
	return err
}

// Create implements catalog.RecipeStore.Create.
func (s *CatalogSQLiteStore) Create(ctx context.Context, recipe *catalog.Recipe) error {
	data, err := json.Marshal(recipe)
	if err != nil {
		return fmt.Errorf("marshal recipe: %w", err)
	}

	verified := 0
	if recipe.Verified {
		verified = 1
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_recipes (id, name, gpu_vendor, verified, data) VALUES (?, ?, ?, ?, ?)
	`, recipe.ID, recipe.Name, recipe.Profile.GPUVendor, verified, string(data))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return catalog.ErrRecipeAlreadyExists
		}
		return fmt.Errorf("insert recipe: %w", err)
	}
	return nil
}

// Get implements catalog.RecipeStore.Get.
func (s *CatalogSQLiteStore) Get(ctx context.Context, id string) (*catalog.Recipe, error) {
	var data string
	err := s.db.QueryRowContext(ctx, `SELECT data FROM catalog_recipes WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, catalog.ErrRecipeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query recipe: %w", err)
	}

	var recipe catalog.Recipe
	if err := json.Unmarshal([]byte(data), &recipe); err != nil {
		return nil, fmt.Errorf("unmarshal recipe: %w", err)
	}
	return &recipe, nil
}

// List implements catalog.RecipeStore.List.
func (s *CatalogSQLiteStore) List(ctx context.Context, filter catalog.RecipeFilter) ([]catalog.Recipe, int, error) {
	// Build WHERE clause from filter.
	var conds []string
	var args []any

	if filter.GPUVendor != "" {
		conds = append(conds, "gpu_vendor = ?")
		args = append(args, filter.GPUVendor)
	}
	if filter.VerifiedOnly {
		conds = append(conds, "verified = 1")
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// Count total matching rows.
	var total int
	countArgs := append([]any{}, args...)
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM catalog_recipes %s`, where), countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count recipes: %w", err)
	}

	// Apply pagination.
	query := fmt.Sprintf(`SELECT data FROM catalog_recipes %s ORDER BY id`, where)
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	} else if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list recipes: %w", err)
	}
	defer rows.Close()

	var recipes []catalog.Recipe
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("scan recipe: %w", err)
		}
		var r catalog.Recipe
		if err := json.Unmarshal([]byte(data), &r); err != nil {
			return nil, 0, fmt.Errorf("unmarshal recipe: %w", err)
		}
		// Apply tag filter in Go (SQLite doesn't handle JSON arrays efficiently).
		if len(filter.Tags) > 0 && !hasAnyTagSQLite(r.Tags, filter.Tags) {
			total--
			continue
		}
		recipes = append(recipes, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate recipes: %w", err)
	}

	return recipes, total, nil
}

// Delete implements catalog.RecipeStore.Delete.
func (s *CatalogSQLiteStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM catalog_recipes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return catalog.ErrRecipeNotFound
	}
	return nil
}

// Update implements catalog.RecipeStore.Update.
func (s *CatalogSQLiteStore) Update(ctx context.Context, recipe *catalog.Recipe) error {
	data, err := json.Marshal(recipe)
	if err != nil {
		return fmt.Errorf("marshal recipe: %w", err)
	}
	verified := 0
	if recipe.Verified {
		verified = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE catalog_recipes SET name = ?, gpu_vendor = ?, verified = ?, data = ? WHERE id = ?
	`, recipe.Name, recipe.Profile.GPUVendor, verified, string(data), recipe.ID)
	if err != nil {
		return fmt.Errorf("update recipe: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return catalog.ErrRecipeNotFound
	}
	return nil
}

func hasAnyTagSQLite(recipeTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(recipeTags))
	for _, t := range recipeTags {
		tagSet[t] = struct{}{}
	}
	for _, t := range filterTags {
		if _, ok := tagSet[t]; ok {
			return true
		}
	}
	return false
}
