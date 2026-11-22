// Command upgrade converts a database from an older HomeLog so that the
// current version can start on it. Stop HomeLog and back up the data folder
// first, run it once, then start the new version, which migrates the rest.
// Running it again changes nothing.
//
// In Docker: docker compose run --rm homelog ./upgrade
//
// --merge-households shows which households have the same people and would be
// merged into one (see merge.go); add --apply to merge them.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/models"
)

func main() {
	merge := flag.Bool("merge-households", false, "show the households that have the same people, to merge them")
	apply := flag.Bool("apply", false, "with --merge-households: merge them")
	flag.Parse()

	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "./data/homelog.db"
	}
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("❌ No database at %s: %v", path, err)
	}
	db, err := database.Open(path, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalf("❌ Cannot open %s: %v", path, err)
	}
	if !*merge {
		if err := upgrade(db); err != nil {
			log.Fatalf("❌ Upgrade failed, the database is as it was before this step: %v", err)
		}
		hintHouseholdMerges(db)
		return
	}
	if len(database.PendingUpgrades(db)) > 0 {
		log.Fatal("❌ Run ./upgrade without options first.")
	}
	plans, err := planHouseholdMerges(db)
	if err != nil {
		log.Fatalf("❌ Cannot plan the merge: %v", err)
	}
	printMergePlans(os.Stdout, plans)
	if !*apply {
		if len(plans) > 0 {
			fmt.Println("Nothing changed. Back up the data folder, then run again with --apply to merge.")
		}
		return
	}
	if err := applyMergePlans(db, plans); err != nil {
		log.Fatalf("❌ %v", err)
	}
	log.Println("✅ Households merged.")
}

// hintHouseholdMerges tells, after an upgrade, whether some households could
// be merged. It never merges by itself: keeping them apart may be the point.
func hintHouseholdMerges(db *gorm.DB) {
	plans, err := planHouseholdMerges(db)
	if err != nil || len(plans) == 0 {
		return
	}
	log.Printf("ℹ️  %d households have the same people as another one. To see how they would be merged: ./upgrade --merge-households", len(plans))
}

func upgrade(db *gorm.DB) error {
	pending := database.PendingUpgrades(db)
	if len(pending) == 0 {
		log.Println("✅ Nothing to upgrade: this database is ready for the current HomeLog.")
		return nil
	}
	// Rebuilding a table fails while other tables reference it.
	err := database.WithoutForeignKeys(db, func(conn *gorm.DB) error {
		for _, step := range []struct {
			name string
			run  func(*gorm.DB) error
		}{
			// Before the default template goes: it tells which property a template serves.
			{"bill templates moved to their property", moveBillTemplatesToProperty},
			{"bill estimates moved to the closing reading", moveEstimatesToClosingReading},
			{"per-service default template removed", dropServiceDefaultTemplate},
			// Last: the steps above still read household_members.property_id.
			{"homes grouped into households", groupHomesIntoHouseholds},
		} {
			if err := step.run(conn); err != nil {
				return fmt.Errorf("%s: %w", step.name, err)
			}
			log.Printf("✅ %s", step.name)
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Println("✅ Done. Start HomeLog: it migrates the rest on startup.")
	return nil
}

// moveBillTemplatesToProperty gives every per-user bill template the property
// it serves, then makes property_id NOT NULL. The property is, in order: the
// one of a service using the template as its default; the one of a service of
// the same type and provider among the author's households; the author's
// current property; the author's first household. A template matching none was
// usable by nobody and is deleted. A service of another property that used the
// template as its default gets its own copy.
func moveBillTemplatesToProperty(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("bill_templates") || m.HasColumn("bill_templates", "property_id") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		exec := func(stmts ...string) error {
			for _, stmt := range stmts {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return nil
		}
		if err := exec(
			`ALTER TABLE bill_templates ADD COLUMN property_id integer`,
			`UPDATE bill_templates SET property_id = COALESCE(
				(SELECT u.property_id FROM utilities u
					WHERE u.default_bill_template_id = bill_templates.id
					ORDER BY u.deleted_at IS NOT NULL, u.id LIMIT 1),
				(SELECT u.property_id FROM utilities u
					JOIN household_members hm ON hm.property_id = u.property_id AND hm.deleted_at IS NULL
					WHERE hm.user_id = bill_templates.user_id AND u.deleted_at IS NULL
					AND u.type = bill_templates.utility_type
					AND LOWER(TRIM(u.provider)) = LOWER(TRIM(bill_templates.provider))
					ORDER BY u.id LIMIT 1),
				(SELECT p.id FROM properties p
					JOIN household_members hm ON hm.property_id = p.id AND hm.deleted_at IS NULL
					WHERE hm.user_id = bill_templates.user_id AND p.is_current = 1 AND p.deleted_at IS NULL
					ORDER BY p.id LIMIT 1),
				(SELECT hm.property_id FROM household_members hm
					WHERE hm.user_id = bill_templates.user_id AND hm.deleted_at IS NULL
					ORDER BY hm.id LIMIT 1))`,
			`DELETE FROM bill_templates WHERE property_id IS NULL`,
		); err != nil {
			return err
		}

		var crossed []struct{ UtilityID, PropertyID, TemplateID uint }
		if err := tx.Raw(`SELECT u.id AS utility_id, u.property_id, t.id AS template_id
			FROM utilities u JOIN bill_templates t ON t.id = u.default_bill_template_id
			WHERE u.property_id != t.property_id`).Scan(&crossed).Error; err != nil {
			return err
		}
		copies := map[[2]uint]uint{}
		for _, x := range crossed {
			key := [2]uint{x.TemplateID, x.PropertyID}
			if copies[key] == 0 {
				if err := tx.Exec(`INSERT INTO bill_templates
					(created_at, updated_at, user_id, name, provider, utility_type, is_default, extraction_rules, property_id)
					SELECT created_at, updated_at, user_id, name, provider, utility_type, is_default, extraction_rules, ?
					FROM bill_templates WHERE id = ?`, x.PropertyID, x.TemplateID).Error; err != nil {
					return err
				}
				var id uint
				if err := tx.Raw(`SELECT last_insert_rowid()`).Scan(&id).Error; err != nil {
					return err
				}
				copies[key] = id
			}
			if err := tx.Exec(`UPDATE utilities SET default_bill_template_id = ? WHERE id = ?`, copies[key], x.UtilityID).Error; err != nil {
				return err
			}
		}

		if err := exec(
			// Defaults were unique per author; keep the newest one per property.
			`UPDATE bill_templates SET is_default = 0 WHERE is_default = 1 AND EXISTS (
				SELECT 1 FROM bill_templates b WHERE b.is_default = 1 AND b.id > bill_templates.id
				AND b.property_id = bill_templates.property_id AND b.utility_type = bill_templates.utility_type
				AND LOWER(TRIM(b.provider)) = LOWER(TRIM(bill_templates.provider)))`,
		); err != nil {
			return err
		}
		// AutoMigrate relaxes a column to NULL but never tightens one to NOT NULL.
		return tx.Migrator().AlterColumn(&models.BillTemplate{}, "PropertyID")
	})
}

// moveEstimatesToClosingReading makes a bill's estimate its closing reading,
// where the bill ends and the next one re-bills from, and maps template rules
// the same way; then it drops the estimate columns.
func moveEstimatesToClosingReading(db *gorm.DB) error {
	if !db.Migrator().HasColumn("bills", "estimated_reading") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range []string{
			`UPDATE bills SET provider_reading = estimated_reading,
				provider_reading_date = COALESCE(estimated_date, provider_reading_date),
				reading_type = 'estimated'
			WHERE estimated_reading IS NOT NULL`,
			`ALTER TABLE bills DROP COLUMN estimated_date`,
			`ALTER TABLE bills DROP COLUMN estimated_reading`,
			`ALTER TABLE bills DROP COLUMN estimated_consumption`,
		} {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}

		var templates []models.BillTemplate
		if err := tx.Where("extraction_rules LIKE ?", `%"estimated_%`).Find(&templates).Error; err != nil {
			return err
		}
		renamed := map[string]string{"estimated_reading": "provider_reading", "estimated_date": "provider_reading_date"}
		for _, tpl := range templates {
			var rules []map[string]any
			if err := json.Unmarshal([]byte(tpl.ExtractionRules), &rules); err != nil {
				continue
			}
			mapped := map[string]bool{}
			for _, r := range rules {
				if to, ok := renamed[fmt.Sprint(r["field"])]; ok {
					mapped[to] = true
				}
			}
			kept := rules[:0]
			for _, r := range rules {
				field := fmt.Sprint(r["field"])
				if to, ok := renamed[field]; ok {
					r["field"] = to
				} else if mapped[field] {
					continue
				}
				kept = append(kept, r)
			}
			raw, err := json.Marshal(kept)
			if err != nil {
				return err
			}
			if err := tx.Model(&tpl).UpdateColumn("extraction_rules", string(raw)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// dropServiceDefaultTemplate removes the per-service default template: the
// default is the flag on the template. Dropping the foreign key rebuilds the
// table and loses its indexes, which the server's startup migration restores.
func dropServiceDefaultTemplate(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasColumn("utilities", "default_bill_template_id") {
		return nil
	}
	db.Exec(`DROP INDEX IF EXISTS idx_utilities_default_bill_template_id`)
	if err := m.DropConstraint(&models.Utility{}, "fk_utilities_default_bill_template"); err != nil {
		return err
	}
	// Not m.DropColumn: it finds only the columns GORM itself wrote.
	return db.Exec(`ALTER TABLE utilities DROP COLUMN default_bill_template_id`).Error
}

// groupHomesIntoHouseholds gives every home its own household, named after it
// and with the same id, so existing data behaves exactly as before. Members,
// settlements, split settings and join requests move from the home to the
// household; every expense gets a household, and keeps its home only in the
// Home category, the one where a home means something.
func groupHomesIntoHouseholds(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("household_members") || m.HasColumn("household_members", "household_id") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Migrator().CreateTable(&models.Household{}); err != nil {
			return err
		}
		if err := tx.Migrator().CreateTable(&models.HouseholdJoinRequest{}); err != nil {
			return err
		}
		for _, stmt := range []string{
			`INSERT INTO households (id, created_at, updated_at, deleted_at, name)
				SELECT id, created_at, updated_at, deleted_at, name FROM properties`,
			`ALTER TABLE properties ADD COLUMN household_id integer`,
			`UPDATE properties SET household_id = id`,
			// Primary was one home per owner; it is one per household now, and
			// every household starts with a single home.
			`UPDATE properties SET is_current = 1`,
			`ALTER TABLE household_members ADD COLUMN household_id integer`,
			`UPDATE household_members SET household_id = property_id`,
			`ALTER TABLE settlements ADD COLUMN household_id integer`,
			`UPDATE settlements SET household_id = property_id`,
			`ALTER TABLE household_settings ADD COLUMN household_id integer`,
			`UPDATE household_settings SET household_id = property_id`,
			`INSERT INTO household_join_requests
				(id, created_at, updated_at, deleted_at, user_id, household_id, status, resolved_by, resolved_at)
				SELECT id, created_at, updated_at, deleted_at, user_id, property_id, status, resolved_by, resolved_at
				FROM property_join_requests`,
			`DROP TABLE property_join_requests`,
			`ALTER TABLE expenses ADD COLUMN household_id integer`,
			`UPDATE expenses SET household_id = property_id`,
			// An expense without a home belongs to its author's first household.
			`UPDATE expenses SET household_id = (
				SELECT hm.household_id FROM household_members hm
				WHERE hm.user_id = expenses.user_id ORDER BY hm.created_at, hm.id LIMIT 1)
			WHERE household_id IS NULL`,
			// The form forced a home on every expense: only Home ones keep it.
			`UPDATE expenses SET property_id = NULL
			WHERE property_id IS NOT NULL
				AND category_id NOT IN (SELECT id FROM categories WHERE slug = 'home')`,
		} {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}

		var homeless int64
		if err := tx.Table("expenses").Where("household_id IS NULL").Count(&homeless).Error; err != nil {
			return err
		}
		if homeless > 0 {
			return fmt.Errorf("%d expenses belong to no home and their authors to no household: assign them a home first", homeless)
		}

		for _, table := range []string{"household_members", "settlements", "household_settings"} {
			if err := database.DropColumns(tx, table, "property_id"); err != nil {
				return fmt.Errorf("drop %s.property_id: %w", table, err)
			}
		}
		return nil
	})
}
