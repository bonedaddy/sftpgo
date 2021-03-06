// +build !nomysql

package dataprovider

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	// we import go-sql-driver/mysql here to be able to disable MySQL support using a build tag
	_ "github.com/go-sql-driver/mysql"

	"github.com/drakkan/sftpgo/logger"
	"github.com/drakkan/sftpgo/utils"
	"github.com/drakkan/sftpgo/vfs"
)

const (
	mysqlUsersTableSQL = "CREATE TABLE `{{users}}` (`id` integer AUTO_INCREMENT NOT NULL PRIMARY KEY, " +
		"`username` varchar(255) NOT NULL UNIQUE, `password` varchar(255) NULL, `public_keys` longtext NULL, " +
		"`home_dir` varchar(255) NOT NULL, `uid` integer NOT NULL, `gid` integer NOT NULL, `max_sessions` integer NOT NULL, " +
		" `quota_size` bigint NOT NULL, `quota_files` integer NOT NULL, `permissions` longtext NOT NULL, " +
		"`used_quota_size` bigint NOT NULL, `used_quota_files` integer NOT NULL, `last_quota_update` bigint NOT NULL, " +
		"`upload_bandwidth` integer NOT NULL, `download_bandwidth` integer NOT NULL, `expiration_date` bigint(20) NOT NULL, " +
		"`last_login` bigint(20) NOT NULL, `status` int(11) NOT NULL, `filters` longtext DEFAULT NULL, " +
		"`filesystem` longtext DEFAULT NULL);"
	mysqlSchemaTableSQL = "CREATE TABLE `{{schema_version}}` (`id` integer AUTO_INCREMENT NOT NULL PRIMARY KEY, `version` integer NOT NULL);"
	mysqlV2SQL          = "ALTER TABLE `{{users}}` ADD COLUMN `virtual_folders` longtext NULL;"
	mysqlV3SQL          = "ALTER TABLE `{{users}}` MODIFY `password` longtext NULL;"
	mysqlV4SQL          = "CREATE TABLE `{{folders}}` (`id` integer AUTO_INCREMENT NOT NULL PRIMARY KEY, `path` varchar(512) NOT NULL UNIQUE," +
		"`used_quota_size` bigint NOT NULL, `used_quota_files` integer NOT NULL, `last_quota_update` bigint NOT NULL);" +
		"ALTER TABLE `{{users}}` MODIFY `home_dir` varchar(512) NOT NULL;" +
		"ALTER TABLE `{{users}}` DROP COLUMN `virtual_folders`;" +
		"CREATE TABLE `{{folders_mapping}}` (`id` integer AUTO_INCREMENT NOT NULL PRIMARY KEY, `virtual_path` varchar(512) NOT NULL, " +
		"`quota_size` bigint NOT NULL, `quota_files` integer NOT NULL, `folder_id` integer NOT NULL, `user_id` integer NOT NULL);" +
		"ALTER TABLE `{{folders_mapping}}` ADD CONSTRAINT `unique_mapping` UNIQUE (`user_id`, `folder_id`);" +
		"ALTER TABLE `{{folders_mapping}}` ADD CONSTRAINT `folders_mapping_folder_id_fk_folders_id` FOREIGN KEY (`folder_id`) REFERENCES `{{folders}}` (`id`) ON DELETE CASCADE;" +
		"ALTER TABLE `{{folders_mapping}}` ADD CONSTRAINT `folders_mapping_user_id_fk_users_id` FOREIGN KEY (`user_id`) REFERENCES `{{users}}` (`id`) ON DELETE CASCADE;"
)

// MySQLProvider auth provider for MySQL/MariaDB database
type MySQLProvider struct {
	dbHandle *sql.DB
}

func init() {
	utils.AddFeature("+mysql")
}

func initializeMySQLProvider() error {
	var err error
	logSender = fmt.Sprintf("dataprovider_%v", MySQLDataProviderName)
	dbHandle, err := sql.Open("mysql", getMySQLConnectionString(false))
	if err == nil {
		providerLog(logger.LevelDebug, "mysql database handle created, connection string: %#v, pool size: %v",
			getMySQLConnectionString(true), config.PoolSize)
		dbHandle.SetMaxOpenConns(config.PoolSize)
		dbHandle.SetConnMaxLifetime(1800 * time.Second)
		provider = MySQLProvider{dbHandle: dbHandle}
	} else {
		providerLog(logger.LevelWarn, "error creating mysql database handler, connection string: %#v, error: %v",
			getMySQLConnectionString(true), err)
	}
	return err
}
func getMySQLConnectionString(redactedPwd bool) string {
	var connectionString string
	if len(config.ConnectionString) == 0 {
		password := config.Password
		if redactedPwd {
			password = "[redacted]"
		}
		connectionString = fmt.Sprintf("%v:%v@tcp([%v]:%v)/%v?charset=utf8&interpolateParams=true&timeout=10s&tls=%v&writeTimeout=10s&readTimeout=10s",
			config.Username, password, config.Host, config.Port, config.Name, getSSLMode())
	} else {
		connectionString = config.ConnectionString
	}
	return connectionString
}

func (p MySQLProvider) checkAvailability() error {
	return sqlCommonCheckAvailability(p.dbHandle)
}

func (p MySQLProvider) validateUserAndPass(username string, password string) (User, error) {
	return sqlCommonValidateUserAndPass(username, password, p.dbHandle)
}

func (p MySQLProvider) validateUserAndPubKey(username string, publicKey []byte) (User, string, error) {
	return sqlCommonValidateUserAndPubKey(username, publicKey, p.dbHandle)
}

func (p MySQLProvider) getUserByID(ID int64) (User, error) {
	return sqlCommonGetUserByID(ID, p.dbHandle)
}

func (p MySQLProvider) updateQuota(username string, filesAdd int, sizeAdd int64, reset bool) error {
	return sqlCommonUpdateQuota(username, filesAdd, sizeAdd, reset, p.dbHandle)
}

func (p MySQLProvider) getUsedQuota(username string) (int, int64, error) {
	return sqlCommonGetUsedQuota(username, p.dbHandle)
}

func (p MySQLProvider) updateLastLogin(username string) error {
	return sqlCommonUpdateLastLogin(username, p.dbHandle)
}

func (p MySQLProvider) userExists(username string) (User, error) {
	return sqlCommonCheckUserExists(username, p.dbHandle)
}

func (p MySQLProvider) addUser(user User) error {
	return sqlCommonAddUser(user, p.dbHandle)
}

func (p MySQLProvider) updateUser(user User) error {
	return sqlCommonUpdateUser(user, p.dbHandle)
}

func (p MySQLProvider) deleteUser(user User) error {
	return sqlCommonDeleteUser(user, p.dbHandle)
}

func (p MySQLProvider) dumpUsers() ([]User, error) {
	return sqlCommonDumpUsers(p.dbHandle)
}

func (p MySQLProvider) getUsers(limit int, offset int, order string, username string) ([]User, error) {
	return sqlCommonGetUsers(limit, offset, order, username, p.dbHandle)
}

func (p MySQLProvider) dumpFolders() ([]vfs.BaseVirtualFolder, error) {
	return sqlCommonDumpFolders(p.dbHandle)
}

func (p MySQLProvider) getFolders(limit, offset int, order, folderPath string) ([]vfs.BaseVirtualFolder, error) {
	return sqlCommonGetFolders(limit, offset, order, folderPath, p.dbHandle)
}

func (p MySQLProvider) getFolderByPath(mappedPath string) (vfs.BaseVirtualFolder, error) {
	return sqlCommonCheckFolderExists(mappedPath, p.dbHandle)
}

func (p MySQLProvider) addFolder(folder vfs.BaseVirtualFolder) error {
	return sqlCommonAddFolder(folder, p.dbHandle)
}

func (p MySQLProvider) deleteFolder(folder vfs.BaseVirtualFolder) error {
	return sqlCommonDeleteFolder(folder, p.dbHandle)
}

func (p MySQLProvider) updateFolderQuota(mappedPath string, filesAdd int, sizeAdd int64, reset bool) error {
	return sqlCommonUpdateFolderQuota(mappedPath, filesAdd, sizeAdd, reset, p.dbHandle)
}

func (p MySQLProvider) getUsedFolderQuota(mappedPath string) (int, int64, error) {
	return sqlCommonGetFolderUsedQuota(mappedPath, p.dbHandle)
}

func (p MySQLProvider) close() error {
	return p.dbHandle.Close()
}

func (p MySQLProvider) reloadConfig() error {
	return nil
}

// initializeDatabase creates the initial database structure
func (p MySQLProvider) initializeDatabase() error {
	sqlUsers := strings.Replace(mysqlUsersTableSQL, "{{users}}", sqlTableUsers, 1)
	tx, err := p.dbHandle.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlUsers)
	if err != nil {
		sqlCommonRollbackTransaction(tx)
		return err
	}
	_, err = tx.Exec(strings.Replace(mysqlSchemaTableSQL, "{{schema_version}}", sqlTableSchemaVersion, 1))
	if err != nil {
		sqlCommonRollbackTransaction(tx)
		return err
	}
	_, err = tx.Exec(strings.Replace(initialDBVersionSQL, "{{schema_version}}", sqlTableSchemaVersion, 1))
	if err != nil {
		sqlCommonRollbackTransaction(tx)
		return err
	}
	return tx.Commit()
}

func (p MySQLProvider) migrateDatabase() error {
	dbVersion, err := sqlCommonGetDatabaseVersion(p.dbHandle)
	if err != nil {
		return err
	}
	if dbVersion.Version == sqlDatabaseVersion {
		providerLog(logger.LevelDebug, "sql database is updated, current version: %v", dbVersion.Version)
		return nil
	}
	switch dbVersion.Version {
	case 1:
		err = updateMySQLDatabaseFrom1To2(p.dbHandle)
		if err != nil {
			return err
		}
		err = updateMySQLDatabaseFrom2To3(p.dbHandle)
		if err != nil {
			return err
		}
		return updateMySQLDatabaseFrom3To4(p.dbHandle)
	case 2:
		err = updateMySQLDatabaseFrom2To3(p.dbHandle)
		if err != nil {
			return err
		}
		return updateMySQLDatabaseFrom3To4(p.dbHandle)
	case 3:
		return updateMySQLDatabaseFrom3To4(p.dbHandle)
	default:
		return fmt.Errorf("Database version not handled: %v", dbVersion.Version)
	}
}

func updateMySQLDatabaseFrom1To2(dbHandle *sql.DB) error {
	providerLog(logger.LevelInfo, "updating database version: 1 -> 2")
	sql := strings.Replace(mysqlV2SQL, "{{users}}", sqlTableUsers, 1)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 2)
}

func updateMySQLDatabaseFrom2To3(dbHandle *sql.DB) error {
	providerLog(logger.LevelInfo, "updating database version: 2 -> 3")
	sql := strings.Replace(mysqlV3SQL, "{{users}}", sqlTableUsers, 1)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 3)
}

func updateMySQLDatabaseFrom3To4(dbHandle *sql.DB) error {
	return sqlCommonUpdateDatabaseFrom3To4(mysqlV4SQL, dbHandle)
}
