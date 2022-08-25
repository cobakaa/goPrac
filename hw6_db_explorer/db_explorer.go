package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type DbHandler struct {
	DB            *sql.DB
	ValidationMap map[string]map[string]FieldInfo
}

type Item map[string]interface{}

type SqlError struct {
	HTTPStatus int
	Err        error
}

type FieldInfo struct {
	Field    string
	Type     string
	Nullable bool
	Key      string
	Default  sql.NullString
	Extra    string
}

func (ae SqlError) Error() string {
	return ae.Err.Error()
}

type Response struct {
	Error string                 `json:"error,omitempty"`
	Resp  map[string]interface{} `json:"response,omitempty"`
}

func TablesList(db *sql.DB) ([]string, error) {
	tables, err := db.Query(`SHOW TABLES;`)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	defer tables.Close()
	for tables.Next() {
		name := ""
		err := tables.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func TableFieldsList(db *sql.DB, table string) ([]FieldInfo, error) {
	recs, err := db.Query(fmt.Sprintf("DESCRIBE %s;", table))
	if err != nil {
		return nil, err
	}

	fields := make([]FieldInfo, 0)
	defer recs.Close()
	for recs.Next() {
		field := FieldInfo{}
		nullable := ""
		err := recs.Scan(&field.Field, &field.Type, &nullable, &field.Key, &field.Default, &field.Extra)
		if err != nil {
			return nil, err
		}
		if nullable == "NO" {
			field.Nullable = false
		} else if nullable == "YES" {
			field.Nullable = true
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func (db *DbHandler) RecordsList(table string, limit, offset int) ([]Item, error) {
	q := ""
	if !db.IsTableCorrect(table) {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("unknown table")}
	} else {
		q = fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?;", table)
	}
	records, err := db.DB.Query(q, limit, offset)
	if err != nil {
		panic(err)
	}
	cols := make([]interface{}, len(db.ValidationMap[table]))
	colsPtrs := make([]interface{}, len(db.ValidationMap[table]))
	//colsTypes, _ := records.ColumnTypes()
	//fmt.Println(colsTypes)
	columns, _ := records.Columns()
	for i, _ := range cols {
		colsPtrs[i] = &cols[i]
	}
	items := make([]Item, 0)
	for records.Next() {
		item := make(Item, 0)
		err = records.Scan(colsPtrs...)
		for i, col := range columns {
			//val := colsPtrs[i]
			switch cols[i].(type) {
			case []uint8:
				//val := cols[i]
				item[col] = string(cols[i].([]uint8))
			default:
				item[col] = cols[i]
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (db *DbHandler) IsTableCorrect(table string) bool {
	_, ok := db.ValidationMap[table]
	return ok
}

func (db *DbHandler) RecordById(table string, id string) (Item, error) {

	q := ""
	field := ""
	if !db.IsTableCorrect(table) {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("unknown table")}
	} else {
		for _, s := range db.ValidationMap[table] {
			if strings.Contains(s.Field, "id") {
				field = s.Field
				break
			}
		}
		q = fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;", table, field)
	}
	cols := make([]interface{}, len(db.ValidationMap[table]))
	colsPtrs := make([]interface{}, len(db.ValidationMap[table]))
	for i := range cols {
		colsPtrs[i] = &cols[i]
	}
	record, _ := db.DB.Query(q, id)
	columns, _ := record.Columns()
	record.Next()
	err := record.Scan(colsPtrs...)
	record.Close()
	if err != nil {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("record not found")}
	}
	item := make(Item, 0)
	for i, colName := range columns {
		switch cols[i].(type) {
		case []uint8:
			//val := cols[i]
			item[colName] = string(cols[i].([]uint8))
		default:
			item[colName] = cols[i]
		}
	}
	//fmt.Println(item)
	return item, nil
}

func isFieldCorrect(m map[string]FieldInfo, f string) bool {
	_, ok := m[f]
	return ok
}

func (db *DbHandler) CreateRecord(table string, body Item) (Item, error) {
	q := ""
	field := ""
	if !db.IsTableCorrect(table) {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}
	for _, s := range db.ValidationMap[table] {
		if strings.Contains(s.Field, "id") {
			field = s.Field
			break
		}
	}
	q = fmt.Sprintf("INSERT IGNORE INTO %s (", table)
	bKeys := make([]string, 0)
	bVals := make([]interface{}, 0)
	for k, v := range body {
		if !isFieldCorrect(db.ValidationMap[table], k) {
			continue
		}
		if db.ValidationMap[table][k].Key == "PRI" && db.ValidationMap[table][k].Extra == "auto_increment" {
			continue
		}
		bKeys = append(bKeys, k)
		bVals = append(bVals, v)
	}
	for i, s := range bKeys {
		if i == len(bKeys)-1 {
			q += s + ") VALUES ("
		} else {
			q += s + ", "
		}
	}
	for i, _ := range bVals {
		if i != len(bKeys)-1 {
			q += "?, "
		} else {
			q += "?);"
		}
	}
	result, err := db.DB.Exec(q, bVals...)
	if err != nil {
		return nil, SqlError{http.StatusInternalServerError, fmt.Errorf("put exec error")}
	}
	item := make(Item, 0)
	item[field], _ = result.LastInsertId()
	return item, nil
}

func (db *DbHandler) RecordUpdate(table, id string, body Item) (Item, error) {
	q := ""
	if !db.IsTableCorrect(table) {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}
	q = fmt.Sprintf("UPDATE %s SET ", table)
	bKeys := make([]string, 0)
	bVals := make([]interface{}, 0)
	//validated := true
	for k, v := range body {
		if !isFieldCorrect(db.ValidationMap[table], k) {
			return nil, SqlError{http.StatusBadRequest, fmt.Errorf("bad request")}
		}
		if db.ValidationMap[table][k].Key == "PRI" {
			return nil, SqlError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", k)}
		}
		if !db.ValidationMap[table][k].Nullable && v == nil {
			return nil, SqlError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", k)}
		}
		switch v.(type) {
		case int:
			if db.ValidationMap[table][k].Type != "int" {
				return nil, SqlError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", k)}
			}
		case string:
			if db.ValidationMap[table][k].Type != "text" && !strings.Contains(db.ValidationMap[table][k].Type, "varchar") {
				return nil, SqlError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", k)}
			}
		case float32, float64:
			if db.ValidationMap[table][k].Type != "float" {
				return nil, SqlError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", k)}
			}
		}
		bKeys = append(bKeys, k)
		bVals = append(bVals, v)
	}
	for i := range bKeys {
		q += bKeys[i] + " = ?"
		if i != len(bKeys)-1 {
			q += ", "
		}
	}
	field := ""
	for _, s := range db.ValidationMap[table] {
		if strings.Contains(s.Field, "id") {
			field = s.Field
			break
		}
	}
	q += fmt.Sprintf(" WHERE %s = ?;", field)
	bVals = append(bVals, id)
	_, err := db.DB.Exec(q, bVals...)
	if err != nil {
		return nil, SqlError{http.StatusInternalServerError, fmt.Errorf("post exec error")}
	}
	item := make(Item, 0)
	item["updated"] = 1
	return item, nil
}

func (db *DbHandler) RecordDelete(table, id string) (Item, error) {
	q := ""
	if !db.IsTableCorrect(table) {
		return nil, SqlError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}
	field := ""
	for _, s := range db.ValidationMap[table] {
		if strings.Contains(s.Field, "id") {
			field = s.Field
			break
		}
	}
	q = fmt.Sprintf("DELETE FROM %s WHERE %s = ?;", table, field)
	result, err := db.DB.Exec(q, id)
	if err != nil {
		return nil, SqlError{http.StatusInternalServerError, fmt.Errorf("delete sql error")}

	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, SqlError{http.StatusInternalServerError, fmt.Errorf("affected sql error")}
	}
	item := make(Item, 0)
	item["deleted"] = affected
	return item, nil
}

func (db *DbHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := Response{}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == "/" {
			tables := make([]string, 0)
			for k, _ := range db.ValidationMap {
				tables = append(tables, k)
			}

			resp.Resp = make(map[string]interface{}, 0)
			resp.Resp["tables"] = tables
		} else {
			path := r.URL.Path[1:]
			fLimit := r.FormValue("limit")
			limit := 5
			if fLimit != "" {
				var err error
				limit, err = strconv.Atoi(fLimit)
				if err != nil {
					limit = 5
				}
			}
			fOffset := r.FormValue("offset")
			offset := 0
			if fOffset != "" {
				var err error
				offset, err = strconv.Atoi(fOffset)
				if err != nil {
					offset = 0
				}
			}
			ind := strings.Index(path, "/")
			id := ""
			table := ""
			if ind != -1 {
				table = path[:ind]
				id = path[ind+1:]
				result, err := db.RecordById(table, id)
				if err != nil {
					w.WriteHeader(err.(SqlError).HTTPStatus)
					resp.Error = err.Error()
				} else {
					resp.Resp = make(map[string]interface{}, 0)
					resp.Resp["record"] = result
				}
			} else {
				table = path
				result, err := db.RecordsList(table, limit, offset)
				if err != nil {
					w.WriteHeader(err.(SqlError).HTTPStatus)
					resp.Error = err.Error()
				} else {
					resp.Resp = make(map[string]interface{}, 0)
					resp.Resp["records"] = result
				}
			}
		}
	case http.MethodPut:
		path := r.URL.Path[1:]
		if path[len(path)-1] == '/' {
			path = path[:len(path)-1]
		}
		body, _ := ioutil.ReadAll(r.Body)
		item := make(Item, 0)
		_ = json.Unmarshal(body, &item)
		res, err := db.CreateRecord(path, item)
		if err != nil {
			resp.Error = err.Error()
			w.WriteHeader(err.(SqlError).HTTPStatus)
		} else {
			resp.Resp = make(map[string]interface{}, 0)
			resp.Resp = res
		}
	case http.MethodPost:
		path := r.URL.Path[1:]
		idInd := strings.Index(path, "/")
		table := path[:idInd]
		id := path[idInd+1:]
		body, _ := ioutil.ReadAll(r.Body)
		item := make(Item, 0)
		_ = json.Unmarshal(body, &item)
		result, err := db.RecordUpdate(table, id, item)
		if err != nil {
			resp.Error = err.Error()
			w.WriteHeader(err.(SqlError).HTTPStatus)
		} else {
			resp.Resp = make(map[string]interface{}, 0)
			resp.Resp = result
		}
	case http.MethodDelete:
		path := r.URL.Path[1:]
		idInd := strings.Index(path, "/")
		table := path[:idInd]
		id := path[idInd+1:]
		result, err := db.RecordDelete(table, id)
		if err != nil {
			resp.Error = err.Error()
			w.WriteHeader(err.(SqlError).HTTPStatus)
		} else {
			resp.Resp = make(map[string]interface{}, 0)
			resp.Resp = result
		}
	}
	res, _ := json.Marshal(resp)
	fmt.Fprintf(w, string(res))
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	//mux := http.NewServeMux()
	VMap := make(map[string]map[string]FieldInfo, 0)
	tables, _ := TablesList(db)
	for _, t := range tables {
		VMap[t] = make(map[string]FieldInfo, 0)
	}

	for _, t := range tables {
		fields, _ := TableFieldsList(db, t)
		for _, f := range fields {
			VMap[t][f.Field] = f
		}
	}
	dbH := &DbHandler{db, VMap}
	http.Handle("/", dbH)
	return dbH, nil
}
