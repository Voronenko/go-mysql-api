package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Soontao/go-mysql-api/lib"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/gommon/log"
	"gopkg.in/doug-martin/goqu.v4"
	_ "gopkg.in/doug-martin/goqu.v4/adapters/mysql"
)

// MysqlAPI
type MysqlAPI struct {
	connection       *sql.DB
	databaseMetadata *DataBaseMetadata
	sql              *SQL
}

// NewMysqlAPI create new MysqlAPI instance
func NewMysqlAPI(dbURI string) *MysqlAPI {
	newAPI := &MysqlAPI{}
	newAPI.GetConnectionPool(dbURI)
	log.Debugf("connect to mysql with conn_str: %s", dbURI)
	newAPI.databaseMetadata = newAPI.retriveDatabaseMetadata(newAPI.CurrentDatabaseName())
	newAPI.sql = &SQL{goqu.New("mysql", newAPI.connection), newAPI.databaseMetadata}
	return newAPI
}

// Connection return
func (api *MysqlAPI) Connection() *sql.DB {
	return api.connection
}

// SQL instance
func (api *MysqlAPI) SQL() *SQL {
	return api.sql
}

// GetDatabaseMetadata return database meta
func (api *MysqlAPI) GetDatabaseMetadata() *DataBaseMetadata {
	return api.databaseMetadata
}

// GetConnectionPool which Pool is Singleton Connection Pool
func (api *MysqlAPI) GetConnectionPool(dbURI string) *sql.DB {
	if api.connection == nil {
		pool, err := sql.Open("mysql", dbURI)
		if err != nil {
			log.Fatal(err.Error())
		}
		// 3 minutes unused connections will be closed
		pool.SetConnMaxLifetime(3 * time.Minute)
		pool.SetMaxIdleConns(3)
		pool.SetMaxOpenConns(10)
		api.connection = pool
	}
	return api.connection
}

// Stop MysqlAPI, clean connections
func (api *MysqlAPI) Stop() *MysqlAPI {
	if api.connection != nil {
		api.connection.Close()
	}
	return api
}

// CurrentDatabaseName return current database
func (api *MysqlAPI) CurrentDatabaseName() string {
	rows, err := api.connection.Query("select database()")
	processIfError(err)
	var res string
	for rows.Next() {
		if err := rows.Scan(&res); err != nil {
			log.Fatal(err)
		}
	}
	return res
}

func (api *MysqlAPI) retriveDatabaseMetadata(databaseName string) *DataBaseMetadata {
	var tableMetas []*TableMetadata
	rs := &DataBaseMetadata{DatabaseName: databaseName}
	rows, err := api.connection.Query("show tables")
	processIfError(err)
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		processIfError(err)
		tableMetas = append(tableMetas, api.retriveTableMetadata(tableName))
	}
	rs.Tables = tableMetas
	return rs
}

func processIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (api *MysqlAPI) retriveTableMetadata(tableName string) *TableMetadata {
	rs := &TableMetadata{TableName: tableName}
	var columnMetas []*ColumnMetadata
	rows, err := api.connection.Query(fmt.Sprintf("desc %s", tableName))
	processIfError(err)
	for rows.Next() {
		var columnName, columnType, nullAble, key, defaultValue, extra sql.NullString
		err := rows.Scan(&columnName, &columnType, &nullAble, &key, &defaultValue, &extra)
		processIfError(err)
		columnMeta := &ColumnMetadata{columnName.String, columnType.String, nullAble.String, key.String, defaultValue.String, extra.String}
		columnMetas = append(columnMetas, columnMeta)
	}
	rs.Columns = columnMetas
	return rs
}

// Query by sql
func (api *MysqlAPI) query(sql string, args ...interface{}) ([]map[string]interface{}, error) {
	var rs []map[string]interface{}
	lib.Logger.Debugf("query sql: '%s'", sql)
	rows, err := api.connection.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	// mysql driver not implement rows.ColumnTypes
	cols, _ := rows.Columns()
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		for i, colName := range cols {
			// Yap! Any integer based type will use int type
			// Other type will convert to string, include decimal, date and others
			colV := *columnPointers[i].(*interface{})
			switch (colV).(type) {
			case int64:
				colV = colV.(int64)
			case []uint8:
				colV = fmt.Sprintf("%s", colV)
			}
			m[colName] = colV
		}
		rs = append(rs, m)
	}
	return rs, nil
}

// Exec a sql
func (api *MysqlAPI) exec(sql string, args ...interface{}) (sql.Result, error) {
	lib.Logger.Debugf("exec sql: '%s'", sql)
	return api.connection.Exec(sql, args...)
}

// Create by table name and obj map
func (api *MysqlAPI) Create(table string, obj map[string]interface{}) (rs sql.Result, err error) {
	sql, err := api.sql.InsertByTable(table, obj)
	if err != nil {
		return
	}
	return api.exec(sql)
}

// Update by table name and obj map
func (api *MysqlAPI) Update(table string, id interface{}, obj map[string]interface{}) (rs sql.Result, err error) {
	if id != nil {
		sql, err := api.sql.UpdateByTableAndId(table, id, obj)
		if err != nil {
			return nil, err
		}
		return api.exec(sql)
	} else {
		err = fmt.Errorf("not support update by where")
		return
	}
}

// Delete by table name and where obj
func (api *MysqlAPI) Delete(table string, id interface{}, obj map[string]interface{}) (rs sql.Result, err error) {
	var sSQL string
	if id != nil {
		sSQL, err = api.sql.DeleteByTableAndId(table, id)
	} else {
		sSQL, err = api.sql.DeleteByTable(table, obj)
	}
	if err != nil {
		return
	}
	return api.exec(sSQL)
}

// Select by table name , where or id
func (api *MysqlAPI) Select(table string, id interface{}, limit int, offset int, fields []interface{}, wheres map[string]goqu.Op, links []interface{}) (rs []map[string]interface{}, err error) {
	var sql string
	for _, f := range fields {
		if !api.databaseMetadata.TableHaveField(table, f.(string)) {
			err = fmt.Errorf("table '%s' not have '%s' field !/n", table, f)
			return
		}
	}
	opt := QueryOption{limit: limit, offset: offset, fields: fields, wheres: wheres, links: links}
	if id != nil {
		sql, err = api.sql.GetByTableAndID(table, id, opt)
	} else {
		sql, err = api.sql.GetByTable(table, opt)
	}
	if err != nil {
		return
	}
	return api.query(sql)
}
