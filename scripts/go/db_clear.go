package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 解析命令行参数
	dbPath := flag.String("db", "data/conversations.db", "Database file path")
	mode := flag.String("mode", "", "Operation mode: rebuild | clear-source")
	sourceType := flag.String("source", "", "Source type to clear (required for clear-source mode)")
	flag.Parse()

	// 验证参数
	if *mode == "" {
		log.Fatal("Error: -mode is required (rebuild | clear-source)")
	}

	if *mode == "clear-source" && *sourceType == "" {
		log.Fatal("Error: -source is required for clear-source mode")
	}

	// 转换为绝对路径
	absDBPath, err := filepath.Abs(*dbPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for db: %v", err)
	}

	switch *mode {
	case "rebuild":
		if err := rebuildDatabase(absDBPath); err != nil {
			log.Fatalf("Failed to rebuild database: %v", err)
		}
		log.Println("✓ Database rebuilt successfully")

	case "clear-source":
		if err := clearSourceData(absDBPath, *sourceType); err != nil {
			log.Fatalf("Failed to clear source data: %v", err)
		}
		log.Printf("✓ Cleared all data for source: %s", *sourceType)

	default:
		log.Fatalf("Unknown mode: %s (use rebuild or clear-source)", *mode)
	}
}

// rebuildDatabase 重建数据库
func rebuildDatabase(dbPath string) error {
	log.Printf("Rebuilding database: %s", dbPath)

	// 1. 删除现有数据库文件
	if _, err := os.Stat(dbPath); err == nil {
		log.Println("Deleting existing database file...")
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("remove database file: %w", err)
		}
		log.Println("✓ Database file deleted")
	}

	// 2. 删除 WAL 和 SHM 文件
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"
	os.Remove(walPath)
	os.Remove(shmPath)

	// 3. 查找 init_database.sql
	sqlPath, err := findInitSQL()
	if err != nil {
		return fmt.Errorf("find init sql: %w", err)
	}

	log.Printf("Using init SQL: %s", sqlPath)

	// 4. 读取 SQL 脚本
	sqlBytes, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("read init sql: %w", err)
	}

	// 5. 创建并初始化数据库
	log.Println("Creating and initializing database...")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// 执行初始化脚本
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("execute init sql: %w", err)
	}

	log.Println("✓ Database initialized")
	return nil
}

// clearSourceData 清空指定来源的数据
func clearSourceData(dbPath string, sourceType string) error {
	log.Printf("Clearing data for source: %s", sourceType)

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 统计要删除的数据
	var convCount, msgCount, fragCount, favCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM conversations WHERE source_type = ?", sourceType).Scan(&convCount); err != nil {
		return fmt.Errorf("count conversations: %w", err)
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM messages WHERE conversation_uuid IN (SELECT uuid FROM conversations WHERE source_type = ?)", sourceType).Scan(&msgCount); err != nil {
		return fmt.Errorf("count messages: %w", err)
	}
	if err := tx.QueryRow("SELECT COUNT(*) FROM fragments WHERE conversation_uuid IN (SELECT uuid FROM conversations WHERE source_type = ?)", sourceType).Scan(&fragCount); err != nil {
		return fmt.Errorf("count fragments: %w", err)
	}

	log.Printf("Found: %d conversations, %d messages, %d fragments", convCount, msgCount, fragCount)

	if convCount == 0 {
		log.Printf("No data found for source: %s", sourceType)
		return nil
	}

	// 1. 删除 favorites 表中相关的收藏
	// 删除 target_type='conversation' 的收藏
	result, err := tx.Exec(`
		DELETE FROM favorites
		WHERE target_type = 'conversation'
		  AND target_id IN (SELECT uuid FROM conversations WHERE source_type = ?)
	`, sourceType)
	if err != nil {
		return fmt.Errorf("delete conversation favorites: %w", err)
	}
	if count, _ := result.RowsAffected(); count > 0 {
		favCount += int(count)
	}

	// 删除 target_type='round' 的收藏
	// round 的 target_id 格式为 conversation_uuid-round_index
	result, err = tx.Exec(`
		DELETE FROM favorites
		WHERE target_type = 'round'
		  AND SUBSTR(target_id, 1, INSTR(target_id, '-') - 1) IN (
			  SELECT uuid FROM conversations WHERE source_type = ?
		  )
	`, sourceType)
	if err != nil {
		return fmt.Errorf("delete round favorites: %w", err)
	}
	if count, _ := result.RowsAffected(); count > 0 {
		favCount += int(count)
	}

	// 删除 target_type='message' 的收藏
	result, err = tx.Exec(`
		DELETE FROM favorites
		WHERE target_type = 'message'
		  AND target_id IN (
			  SELECT uuid FROM messages
			  WHERE conversation_uuid IN (SELECT uuid FROM conversations WHERE source_type = ?)
		  )
	`, sourceType)
	if err != nil {
		return fmt.Errorf("delete message favorites: %w", err)
	}
	if count, _ := result.RowsAffected(); count > 0 {
		favCount += int(count)
	}

	// 删除 target_type='fragment' 的收藏
	result, err = tx.Exec(`
		DELETE FROM favorites
		WHERE target_type = 'fragment'
		  AND target_id IN (
			  SELECT uuid FROM fragments
			  WHERE conversation_uuid IN (SELECT uuid FROM conversations WHERE source_type = ?)
		  )
	`, sourceType)
	if err != nil {
		return fmt.Errorf("delete fragment favorites: %w", err)
	}
	if count, _ := result.RowsAffected(); count > 0 {
		favCount += int(count)
	}

	if favCount > 0 {
		log.Printf("✓ Deleted %d favorites", favCount)
	}

	// 2. 删除 conversations（级联删除会自动处理 messages、fragments、conversation_tags）
	log.Printf("Deleting conversations and related data (cascade)...")
	result, err = tx.Exec("DELETE FROM conversations WHERE source_type = ?", sourceType)
	if err != nil {
		return fmt.Errorf("delete conversations: %w", err)
	}

	deletedConv, _ := result.RowsAffected()
	log.Printf("✓ Deleted %d conversations", deletedConv)
	log.Printf("✓ Cascade deleted %d messages", msgCount)
	if fragCount > 0 {
		log.Printf("✓ Cascade deleted %d fragments", fragCount)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// findInitSQL 查找 init_database.sql 文件
func findInitSQL() (string, error) {
	possiblePaths := []string{
		"scripts/init_database.sql",
		"../scripts/init_database.sql",
		"../../scripts/init_database.sql",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("init_database.sql not found in: %v", possiblePaths)
}
