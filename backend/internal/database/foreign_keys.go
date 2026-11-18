package database

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The models declare what deleting a row does to the rows that point at it
// (constraint:OnDelete tags). GORM writes that into a table only when it
// creates the table, and SQLite cannot alter a constraint afterwards, so
// alignForeignKeys brings existing tables in line by rebuilding them.

type foreignKey struct {
	From     string `gorm:"column:from"`
	Table    string `gorm:"column:table"`
	To       string `gorm:"column:to"`
	OnUpdate string `gorm:"column:on_update"`
	OnDelete string `gorm:"column:on_delete"`
}

func foreignKeys(db *gorm.DB, table string) ([]foreignKey, error) {
	var keys []foreignKey
	err := db.Raw(`SELECT "from", "table", "to", on_update, on_delete FROM pragma_foreign_key_list(?)`, table).Scan(&keys).Error
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		return a.From+"\x00"+a.Table+"\x00"+a.OnDelete < b.From+"\x00"+b.Table+"\x00"+b.OnDelete
	})
	return keys, err
}

// referenceSchema is the schema the models produce on an empty database.
func referenceSchema() (*gorm.DB, func(), error) {
	ref, err := Open(":memory:", &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, err
	}
	closeRef := func() {
		if sqlDB, err := ref.DB(); err == nil {
			sqlDB.Close()
		}
	}
	if err := migrateModels(ref); err != nil {
		closeRef()
		return nil, nil, err
	}
	return ref, closeRef, nil
}

// alignForeignKeys rebuilds every existing table whose foreign keys differ
// from the models'. It needs foreign keys off and runs before AutoMigrate, so
// GORM finds the constraints in place and recreates nothing itself.
func alignForeignKeys(conn *gorm.DB) error {
	ref, closeRef, err := referenceSchema()
	if err != nil {
		return fmt.Errorf("build reference schema: %w", err)
	}
	defer closeRef()

	var tables []string
	if err := ref.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`).
		Scan(&tables).Error; err != nil {
		return err
	}
	for _, table := range tables {
		if !conn.Migrator().HasTable(table) {
			continue
		}
		want, err := foreignKeys(ref, table)
		if err != nil {
			return err
		}
		have, err := foreignKeys(conn, table)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(want, have) {
			continue
		}
		log.Printf("🔄 Rebuilding %s: its foreign keys changed", table)
		if err := conn.Transaction(func(tx *gorm.DB) error { return rebuildTable(tx, ref, table, want) }); err != nil {
			return fmt.Errorf("rebuild %s: %w", table, err)
		}
	}
	return nil
}

type column struct {
	Name    string  `gorm:"column:name"`
	Type    string  `gorm:"column:type"`
	Default *string `gorm:"column:dflt_value"`
}

// rebuildTable recreates table with the reference definition, the way SQLite
// documents for any change ALTER TABLE cannot make: create, copy, drop,
// rename, then the indexes again. Columns the models no longer declare are
// kept, nullable. A link declared ON DELETE SET NULL that points nowhere is
// cleared on the way, as the database would have done had it been declared.
func rebuildTable(tx, ref *gorm.DB, table string, keys []foreignKey) error {
	var create string
	if err := ref.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&create).Error; err != nil {
		return err
	}
	staged := table + "__rebuild"
	prefix := "CREATE TABLE `" + table + "`"
	if !strings.HasPrefix(create, prefix) {
		return fmt.Errorf("unexpected definition %q", create)
	}
	create = "CREATE TABLE `" + staged + "`" + strings.TrimPrefix(create, prefix)

	var indexes []string
	if err := tx.Raw(`SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL`, table).
		Scan(&indexes).Error; err != nil {
		return err
	}
	var have, want []column
	if err := tx.Raw(`SELECT name, type, dflt_value FROM pragma_table_info(?)`, table).Scan(&have).Error; err != nil {
		return err
	}
	if err := ref.Raw(`SELECT name, type, dflt_value FROM pragma_table_info(?)`, table).Scan(&want).Error; err != nil {
		return err
	}
	declared := map[string]bool{}
	for _, c := range want {
		declared[c.Name] = true
	}
	unlinkOnDelete := map[string]foreignKey{}
	for _, k := range keys {
		if k.OnDelete == "SET NULL" {
			unlinkOnDelete[k.From] = k
		}
	}

	if err := tx.Exec(create).Error; err != nil {
		return err
	}
	names := make([]string, 0, len(have))
	values := make([]string, 0, len(have))
	for _, c := range have {
		quoted := quote(c.Name)
		if !declared[c.Name] {
			add := "ALTER TABLE " + quote(staged) + " ADD COLUMN " + quoted + " " + c.Type
			if c.Default != nil {
				add += " DEFAULT " + *c.Default
			}
			if err := tx.Exec(add).Error; err != nil {
				return err
			}
			log.Printf("⚠️  %s.%s is not in the models any more: kept as it is", table, c.Name)
		}
		value := quoted
		if k, ok := unlinkOnDelete[c.Name]; ok {
			target := "SELECT " + quote(k.To) + " FROM " + quote(k.Table)
			var dangling int64
			if err := tx.Raw("SELECT count(*) FROM " + quote(table) + " WHERE " + quoted + " IS NOT NULL AND " +
				quoted + " NOT IN (" + target + ")").Scan(&dangling).Error; err != nil {
				return err
			}
			if dangling > 0 {
				log.Printf("⚠️  %s.%s: %d links to missing %s cleared", table, c.Name, dangling, k.Table)
			}
			value = "CASE WHEN " + quoted + " IN (" + target + ") THEN " + quoted + " END"
		}
		names = append(names, quoted)
		values = append(values, value)
	}

	steps := []string{
		"INSERT INTO " + quote(staged) + " (" + strings.Join(names, ", ") + ") SELECT " + strings.Join(values, ", ") + " FROM " + quote(table),
		"DROP TABLE " + quote(table),
		"ALTER TABLE " + quote(staged) + " RENAME TO " + quote(table),
	}
	steps = append(steps, indexes...)
	for _, step := range steps {
		if err := tx.Exec(step).Error; err != nil {
			return err
		}
	}
	return nil
}

func quote(name string) string { return "`" + strings.ReplaceAll(name, "`", "``") + "`" }
