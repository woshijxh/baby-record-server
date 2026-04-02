package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"baby-record-server/internal/auth"
	"baby-record-server/internal/config"
	"baby-record-server/internal/middleware"
	"baby-record-server/internal/repository"
	"baby-record-server/internal/service"
	"baby-record-server/internal/wechat"
	"github.com/gin-gonic/gin"
	driverMySQL "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("load timezone failed: %v", err)
	}

	db, err := openDatabase(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}

	repo := repository.New(db)
	if cfg.AutoMigrate {
		if err := repo.Migrate(context.Background()); err != nil {
			log.Fatalf("auto migrate failed: %v", err)
		}
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	tokenManager := auth.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	wechatClient := wechat.New(cfg.WechatAppID, cfg.WechatAppSecret, cfg.WechatMock)
	svc := service.New(repo, wechatClient, tokenManager, location, cfg.AutoSeedDemoData)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	registerRoutes(router, svc, tokenManager)

	log.Printf("baby-record-server listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func openDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err == nil {
		return db, nil
	}

	var mysqlErr *driverMySQL.MySQLError
	if !errorAsMySQLError(err, &mysqlErr) || mysqlErr.Number != 1049 {
		return nil, err
	}

	if createErr := createDatabase(dsn); createErr != nil {
		return nil, fmt.Errorf("%w; create database failed: %v", err, createErr)
	}

	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func createDatabase(dsn string) error {
	cfg, err := driverMySQL.ParseDSN(dsn)
	if err != nil {
		return err
	}
	dbName := cfg.DBName
	if dbName == "" {
		return fmt.Errorf("database name is empty")
	}

	cfg.DBName = ""
	rootDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer rootDB.Close()

	if err := rootDB.Ping(); err != nil {
		return err
	}

	escaped := strings.ReplaceAll(dbName, "`", "``")
	_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", escaped))
	return err
}

func errorAsMySQLError(err error, target **driverMySQL.MySQLError) bool {
	for err != nil {
		mysqlErr, ok := err.(*driverMySQL.MySQLError)
		if ok {
			*target = mysqlErr
			return true
		}
		type unwrapper interface{ Unwrap() error }
		next, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = next.Unwrap()
	}
	return false
}
